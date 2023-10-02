package profiles

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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

type minimalProfileGuesser struct{}

func (g *minimalProfileGuesser) Guess(
	ctx context.Context,
	dc *dynamic.DynamicClient,
	c *client.Client,
	parameters ...interface{},
) error {
	maybePathOrTargets, ok := parameters[0].(string)
	if !ok {
		return fmt.Errorf("expected a string, got: %v", parameters[0])
	}

	// Check if rule file exists.
	ruleFile, err := filepath.Abs(filepath.Clean(maybePathOrTargets))
	if err != nil {
		return fmt.Errorf("failed to get absolute path of rule file: %w", err)
	}
	if _, err := os.Stat(ruleFile); !os.IsNotExist(err) {
		// Extract metrics from rule file.
		err := extractMetricsFromRuleFile(ruleFile)
		if err != nil {
			return fmt.Errorf("failed to extract metrics from rule file: %w", err)
		}
	} else {
		// Guess the metrics needed to implement minimal collection profile.
		if err := guessMinimalProfileFromTargets(ctx, c, maybePathOrTargets); err != nil {
			return fmt.Errorf("failed to guess minimal profile from targets: %w", err)
		}
	}

	return nil
}

func extractMetricsFromRuleFile(ruleFile string) error {
	ruleGroups, err := rulefmt.ParseFile(ruleFile)
	if err != nil {
		return fmt.Errorf("failed to parse rule file: %v", err)
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
	fmt.Print(toRelabelConfig(fmt.Sprintf("(%s)", strings.Join(metrics.UnsortedList(), "|"))))

	return nil
}

func guessMinimalProfileFromTargets(ctx context.Context, c *client.Client, targets string) error {
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
	fmt.Print(toRelabelConfig(fmt.Sprintf("(%s)", strings.Join(metrics.UnsortedList(), "|"))))

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
