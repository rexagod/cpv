package profiles

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	v1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/model/rulefmt"
	"github.com/prometheus/prometheus/promql/parser"
	"github.com/rexagod/cpv/internal/client"
	"gopkg.in/yaml.v3"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/dynamic"
	"k8s.io/klog/v2"
)

type minimalProfileExtractor struct{}

func (g *minimalProfileExtractor) Extract(
	ctx context.Context,
	dc *dynamic.DynamicClient,
	c *client.Client,
	parameters ...interface{},
) error {
	profile := MinimalCollectionProfile

	maybePathOrTargets, ok := parameters[0].(string)
	if !ok {
		return fmt.Errorf("expected a string, got: %v", parameters[0])
	}

	outputCardinality, ok := parameters[1].(bool)
	if !ok {
		return fmt.Errorf("expected a bool, got: %v", parameters[1])
	}

	// Check if rule file exists.
	ruleFile, err := filepath.Abs(filepath.Clean(maybePathOrTargets))
	if err != nil {
		return fmt.Errorf("failed to get absolute path of rule file: %w", err)
	}
	if _, err := os.Stat(ruleFile); !os.IsNotExist(err) {

		// Extract metrics from rule file.
		err := extractMetricsFromRuleFile(ctx, c, ruleFile, outputCardinality)
		if err != nil {
			return fmt.Errorf("failed to extract metrics from rule file: %w", err)
		}
	} else {

		// Extract the metrics needed to implement minimal collection profile.
		if err := extractMinimalProfileFromTargets(ctx, c, maybePathOrTargets, outputCardinality); err != nil {
			return fmt.Errorf("failed to extract %s profile from targets: %w", profile, err)
		}
	}

	return nil
}

func extractMetricsFromRuleFile(ctx context.Context, c *client.Client, ruleFile string, outputCardinality bool) error {
	ruleGroups, parseErr := rulefmt.ParseFile(ruleFile)
	if parseErr != nil {
		return fmt.Errorf("failed to parse rule file: %v", parseErr)
	}
	metrics := sets.Set[string]{}
	for _, group := range ruleGroups.Groups {
		for _, rule := range group.Rules {
			expr, err := parser.ParseExpr(rule.Expr.Value)
			if err != nil {
				return fmt.Errorf("failed to parse targets: %w", err)
			}
			parser.Inspect(
				expr, func(node parser.Node, path []parser.Node) error {
					if n, ok := node.(*parser.VectorSelector); ok {
						metric := n.Name
						metrics.Insert(metric)
					}

					return nil
				},
			)
		}
	}

	err := handleOutput(ctx, c, metrics, outputCardinality)
	if err != nil {
		return err
	}

	return nil
}

func extractMinimalProfileFromTargets(ctx context.Context, c *client.Client, targets string, outputCardinality bool) error {
	expr, err := parser.ParseExpr(targets)
	if err != nil {
		return fmt.Errorf("failed to parse targets: %w", err)
	}
	didEncounterUnexpectedMatchType := false
	parser.Inspect(
		expr, func(node parser.Node, path []parser.Node) error {
			if n, ok := node.(*parser.VectorSelector); ok {
				for _, lm := range n.LabelMatchers {

					// Only key*=*value matchers are supported.
					if lm.Type != labels.MatchEqual {

						// Errors are not returned so that the traversal is not interrupted.
						// Refer: https://github.com/prometheus/prometheus/blob/main/promql/parser/ast.go#L351.
						klog.Errorf("unexpected match type: %s", lm.Type)
						didEncounterUnexpectedMatchType = true

						// Stop traversing the AST.
						//nolint:wrapcheck
						return err
					}
				}
			}

			return nil
		},
	)
	if didEncounterUnexpectedMatchType {
		return fmt.Errorf("unexpected match type encountered, supported match types are: %s", labels.MatchEqual)
	}
	targetsMetadata, err := c.API.TargetsMetadata(ctx, targets, "", "")
	if err != nil {
		return fmt.Errorf("failed to fetch targets metadata: %w", err)
	}
	metrics := sets.Set[string]{}
	for _, data := range targetsMetadata {
		m := data.Metric
		metrics.Insert(m)
	}

	err = handleOutput(ctx, c, metrics, outputCardinality)
	if err != nil {
		return err
	}

	return nil
}

func handleOutput(ctx context.Context, c *client.Client, metrics sets.Set[string], outputCardinality bool) error {

	// Write cardinality statistics to a file.
	metricSet := metrics.UnsortedList()
	logFile, err := os.CreateTemp("/tmp", fmt.Sprintf("%s-profile-extractor-cardinality-statistics*.log", MinimalCollectionProfile))
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer func() {
		_ = logFile.Close()
	}()
	logRecorder := &Recorder{file: logFile}
	logW := tabwriter.NewWriter(logRecorder, 0, 0, 2, ' ', 0)
	columns := fmt.Sprintf("METRIC\tCARDINALITY\n")
	if outputCardinality {
		cardinalityStats := c.EvaluateCardinalities(ctx, &metrics)
		_, _ = fmt.Fprint(logW, columns)
		for _, cardinalityStat := range cardinalityStats {
			_, _ = fmt.Fprintf(logW, "%s\t%d\n", cardinalityStat.Metric, cardinalityStat.Value)
		}
		klog.Infof("cardinality statistics written, refer: %s", logFile.Name())
	} else {
		_ = os.Remove(logFile.Name())
	}
	_ = logW.Flush()

	// Write the relabel config (with the extracted metrics) to a file.
	relabelConfig := toRelabelConfig(fmt.Sprintf("(%s)", strings.Join(metricSet, "|")))
	relabelConfigFile, err := os.CreateTemp("/tmp", fmt.Sprintf("%s-profile-extractor-relabel-config-*.yaml", MinimalCollectionProfile))
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer func() {
		_ = relabelConfigFile.Close()
	}()
	_, _ = fmt.Fprint(relabelConfigFile, relabelConfig)
	klog.Infof("relabel config written, refer: %s", relabelConfigFile.Name())

	return nil
}

func toRelabelConfig(metricsRegex string) string {
	relabelConfig := v1.RelabelConfig{
		SourceLabels: []v1.LabelName{"__name__"},
		Regex:        metricsRegex,
		Action:       "keep",
	}
	relabelConfigBytes, err := yaml.Marshal(relabelConfig)
	if err != nil {
		klog.Fatalf("failed to marshal relabel config: %v", err)
	}

	return string(relabelConfigBytes)
}
