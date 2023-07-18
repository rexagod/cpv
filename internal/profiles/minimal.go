package profiles

import (
	"context"
	"fmt"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/prometheus/promql/parser"
	"github.com/rexagod/cpv/internal/client"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/dynamic"
	"k8s.io/klog/v2"
	"regexp"
)

func MinimalCollectionProfileOperator(ctx context.Context, profile CollectionProfile, dc *dynamic.DynamicClient, c *client.Client) error {
	// Fetch all monitors for the profile.
	podMonitors, serviceMonitors, err := fetchMonitorsForProfile(ctx, dc, profile)
	if err != nil {
		klog.Errorf("failed to fetch monitors for profile %s: %v", profile, err)
	}

	// `metrics` has all the loaded metrics that are present in the Prometheus instance at `--address`.
	var metrics sets.Set[string]
	targets, err := c.TargetsMetadata(ctx, "", "", "")
	if err != nil {
		return fmt.Errorf("failed to fetch targets metadata: %v", err)
	}
	for _, data := range targets {
		m := data.Metric
		metrics.Insert(m)
	}

	// `rules` has all the rules discovered by the Prometheus instance at `--address`.
	rules, err := c.Rules(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch rules: %v", err)
	}

	// regexps is the collective union of all the profile-defining regexps from all the monitors.
	var regexps sets.Set[string]
	for _, servicemonitor := range serviceMonitors.Items {
		expr := extractMetricsExpressionsFromServiceMonitor(servicemonitor)
		regexps.Union(expr)
	}
	for _, podmonitor := range podMonitors.Items {
		expr := extractMetricsExpressionsFromPodMonitor(podmonitor)
		regexps.Union(expr)
	}

	// Check if the metrics in the rules are loaded.
	// If not, check if they match any of the regexps.
	// If they do, then we have a direct correlation between a rule using a metric that is defined by a profile-specific monitor.
	// This essentially means that the associated profile does not have all the required metrics available at this point of time.
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
				klog.Errorf("unexpected rule type %T", v)
			}
			if q == "" {
				continue
			}
			expr, err := parser.ParseExpr(q)
			if err != nil {
				klog.Errorf("failed to parse query %q: %v", q, err)
				continue
			}
			parser.Inspect(expr, func(node parser.Node, path []parser.Node) error {
				switch n := node.(type) {
				case *parser.VectorSelector:
					// Throw if:
					//  * a metric is present one of the rule files, and,
					//  * it is not loaded...
					if !metrics.Has(n.Name) {
						klog.Warningf("metric not loaded: %s", n.Name)
						klog.Warningf("from source: %s under group: %s", group.File, group.Name)
						klog.Warningf("affecting rule: %s with query: %s", ruleName, q)
						for regex := range regexps {
							match, err := regexp.Match(regex, []byte(n.Name))
							if err != nil {
								klog.Errorf("failed to match regex %q: %v", regex, err)
							}
							// * ...while a profile depends on it.
							if match {
								klog.Warningf("corresponding %s profile includes the aforementioned metric, and my be affected", profile)
							}
						}
					}
				}
				return nil
			})
		}
	}
	return nil
}
