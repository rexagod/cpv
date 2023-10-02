package profiles

import (
	"context"
	"fmt"
	"github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"os"
	"strings"
	"sync"
)

const (
	ErrImplemented  = "not implemented"
	ErrLoaded       = "not loaded"
	ErrNonNilIssues = "encountered %d issues, refer: %s"
)

type Recorder struct {
	file                 *os.File
	m                    sync.Mutex
	loadIssues           *uint
	implementationIssues *uint
}

func (r *Recorder) Write(p []byte) (n int, err error) {
	r.m.Lock()
	defer r.m.Unlock()
	if r.loadIssues != nil && strings.Contains(string(p), ErrLoaded) {
		*r.loadIssues++
	}
	if r.implementationIssues != nil && strings.Contains(string(p), ErrImplemented) {
		*r.implementationIssues++
	}
	return r.file.Write(p)
}

func fetchServiceMonitorsForProfile(ctx context.Context, dc *dynamic.DynamicClient, profile CollectionProfile) (*monitoringv1.ServiceMonitorList, error) {
	labelSelector := CollectionProfileOptInLabel
	if profile != "" {
		labelSelector = labelSelector + "=" + string(profile)
	}
	l, err := dc.Resource(schema.GroupVersionResource{
		Group:    monitoring.GroupName,
		Version:  monitoringv1.Version,
		Resource: monitoringv1.ServiceMonitorName,
	}).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list servicemonitors: %w", err)
	}
	var serviceMonitors *monitoringv1.ServiceMonitorList
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(l.UnstructuredContent(), &serviceMonitors)
	if err != nil {
		return nil, fmt.Errorf("failed to convert unstructured to servicemonitor: %w", err)
	}

	return serviceMonitors, nil
}

func fetchPodMonitorsForProfile(ctx context.Context, dc *dynamic.DynamicClient, profile CollectionProfile) (*monitoringv1.PodMonitorList, error) {
	labelSelector := CollectionProfileOptInLabel
	if profile != "" {
		labelSelector = labelSelector + "=" + string(profile)
	}
	l, err := dc.Resource(schema.GroupVersionResource{
		Group:    monitoring.GroupName,
		Version:  monitoringv1.Version,
		Resource: monitoringv1.PodMonitorName,
	}).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list podmonitors: %w", err)
	}
	var podMonitors *monitoringv1.PodMonitorList
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(l.UnstructuredContent(), &podMonitors)
	if err != nil {
		return nil, fmt.Errorf("failed to convert unstructured to podmonitor: %w", err)
	}

	return podMonitors, nil
}

// fetchMonitorsForProfile returns pod and service monitors that implement the specified profile, leave it out to get
// monitors for all profiles.
func fetchMonitorsForProfile(ctx context.Context, dc *dynamic.DynamicClient, profile CollectionProfile) (*monitoringv1.PodMonitorList, *monitoringv1.ServiceMonitorList, error) {
	podMonitors, err := fetchPodMonitorsForProfile(ctx, dc, profile)
	if err != nil {
		return nil, nil, err
	}
	serviceMonitors, err := fetchServiceMonitorsForProfile(ctx, dc, profile)
	if err != nil {
		return nil, nil, err
	}

	return podMonitors, serviceMonitors, nil
}

func extractMetricsExpressionsFromServiceMonitor(serviceMonitor *monitoringv1.ServiceMonitor) string {
	var metricsExpressions []string
	for _, endpoint := range serviceMonitor.Spec.Endpoints {
		for _, metricRelabelConfig := range endpoint.MetricRelabelConfigs {
			action := metricRelabelConfig.Action
			sourceLabels := metricRelabelConfig.SourceLabels
			if action == "keep" && len(sourceLabels) == 1 && sourceLabels[0] == "__name__" {
				regex := metricRelabelConfig.Regex
				regex, _ = strings.CutPrefix(regex, "(")
				regex, _ = strings.CutSuffix(regex, ")")
				metricsExpressions = append(metricsExpressions, regex)
			}
		}
	}

	return strings.Join(metricsExpressions, "|")
}

func extractMetricsExpressionsFromPodMonitor(podMonitor *monitoringv1.PodMonitor) string {
	var metricsExpressions []string
	for _, endpoint := range podMonitor.Spec.PodMetricsEndpoints {
		for _, metricRelabelConfig := range endpoint.MetricRelabelConfigs {
			action := metricRelabelConfig.Action
			sourceLabels := metricRelabelConfig.SourceLabels
			if action == "keep" && len(sourceLabels) == 1 && sourceLabels[0] == "__name__" {
				regex := metricRelabelConfig.Regex
				regex, _ = strings.CutPrefix(regex, "(")
				regex, _ = strings.CutSuffix(regex, ")")
				metricsExpressions = append(metricsExpressions, regex)
			}
		}
	}

	return strings.Join(metricsExpressions, "|")
}
