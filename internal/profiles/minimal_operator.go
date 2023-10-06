package profiles

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"
	"text/tabwriter"

	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/prometheus/promql/parser"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/dynamic"
	"k8s.io/klog/v2"

	"github.com/rexagod/cpv/internal/client"
)

type minimalProfileOperator struct{}

func (o *minimalProfileOperator) Operator(ctx context.Context, dc *dynamic.DynamicClient, c *client.Client, noisy bool) error {
	// Fetch all monitors for the profile.
	podMonitors, serviceMonitors, err := fetchMonitorsForProfile(ctx, dc, MinimalCollectionProfile, noisy)
	if err != nil {
		klog.Errorf("failed to fetch monitors for profile %s: %v", MinimalCollectionProfile, err)
	}

	// `metrics` has all the loaded metrics that are present in the Prometheus instance at `--address`.
	metrics := sets.Set[string]{}
	targets, err := c.TargetsMetadata(ctx, "", "", "")
	if err != nil {
		return fmt.Errorf("failed to fetch targets metadata: %w", err)
	}
	for _, data := range targets {
		m := data.Metric
		metrics.Insert(m)
	}

	// `rules` has all the rules discovered by the Prometheus instance at `--address`.
	rules, err := c.Rules(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch rules: %w", err)
	}

	// regexps is the collective union of all the profile-defining regexps from all the monitors.
	regexps := map[string]string{}
	for _, servicemonitor := range serviceMonitors.Items {
		expr := extractMetricsExpressionsFromServiceMonitor(servicemonitor)
		regexps[servicemonitor.Name] = expr
	}
	for _, podmonitor := range podMonitors.Items {
		expr := extractMetricsExpressionsFromPodMonitor(podmonitor)
		regexps[podmonitor.Name] = expr
	}

	file, err := os.CreateTemp("/tmp", fmt.Sprintf("%s-profile-operator-metric-discrepancies-*.log", MinimalCollectionProfile))
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer func() {
		_ = file.Close()
	}()
	recorder := &Recorder{file: file, loadIssues: new(uint)}
	w := tabwriter.NewWriter(recorder, 0, 0, 2, ' ', 0)

	// Check if the metrics in the rules are loaded. If not, check if they match any of the regexps. If they do, then we
	// have a direct correlation between a rule using a metric that is defined by a profile-specific monitor. This
	// essentially means that the associated profile does not have all the required metrics available at this point of
	// time.
	columns := fmt.Sprintf("%s MONITOR\tGROUP\tLOCATION\tRULE\tQUERY\tMETRIC\tERROR", strings.ToUpper(string(MinimalCollectionProfile)))
	_, _ = fmt.Fprintln(w, columns)
	for _, group := range rules.Groups {
		for _, rule := range group.Rules {
			var q string
			var ruleName string
			switch v := rule.(type) {
			case v1.RecordingRule:
				q = v.Query
				ruleName = v.Name
			case v1.AlertingRule:
				q = v.Query
				ruleName = v.Name
			default:
				_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n", "", group.Name, "", "", "", fmt.Sprintf("unknown rule type %T", v))
			}
			if q == "" {
				continue
			}
			expr, err := parser.ParseExpr(q)
			if err != nil {
				_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n", "", group.Name, ruleName, q, "", fmt.Sprintf("failed to parse query: %v", err))

				continue
			}
			u := sets.Set[string]{}
			parser.Inspect(expr, func(node parser.Node, path []parser.Node) error {
				if n, ok := node.(*parser.VectorSelector); ok {
					// Throw if:
					//  * a metric is present one of the rule files, and,
					//  * it is not loaded...
					if !u.Has(n.Name) && !metrics.Has(n.Name) {
						for monitor, regex := range regexps {
							match, err := regexp.MatchString(regex, n.Name)
							if err != nil {
								_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n", monitor, group.Name, group.File, ruleName, q, n.Name, fmt.Sprintf("failed to match regex %q: %v", regex, err))
							}
							// * ...while a profile depends on it.
							if match {
								_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n", monitor, group.Name, group.File, ruleName, q, n.Name, ErrLoaded)
							}
						}
					}

					// Do not throw for metrics that occur more than once in the same query, this is verbose and provides no additional insight whatsoever.
					u.Insert(n.Name)
				}

				return nil
			})
		}
	}

	_ = w.Flush()
	if *recorder.loadIssues > 0 {
		klog.Infof("encountered %d issues, refer: %s", *recorder.loadIssues, file.Name())
	} else {
		_ = os.Remove(file.Name())
	}

	return nil
}
