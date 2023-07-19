package profiles

import (
	"context"
	"github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/dynamic"
)

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
		return nil, err
	}
	var serviceMonitors *monitoringv1.ServiceMonitorList
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(l.UnstructuredContent(), &serviceMonitors)
	if err != nil {
		return nil, err
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
		return nil, err
	}
	var podMonitors *monitoringv1.PodMonitorList
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(l.UnstructuredContent(), &podMonitors)
	if err != nil {
		return nil, err
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

func extractMetricsExpressionsFromServiceMonitor(serviceMonitor *monitoringv1.ServiceMonitor) sets.Set[string] {
	var metricsExpressions sets.Set[string]
	for _, endpoint := range serviceMonitor.Spec.Endpoints {
		for _, metricRelabelConfig := range endpoint.MetricRelabelConfigs {
			action := metricRelabelConfig.Action
			sourceLabels := metricRelabelConfig.SourceLabels
			if action == "keep" && len(sourceLabels) == 1 && sourceLabels[0] == "__name__" {
				metricsExpressions.Insert(metricRelabelConfig.Regex)
			}
		}
	}
	return metricsExpressions
}

func extractMetricsExpressionsFromPodMonitor(podMonitor *monitoringv1.PodMonitor) sets.Set[string] {
	var metricsExpressions sets.Set[string]
	for _, endpoint := range podMonitor.Spec.PodMetricsEndpoints {
		for _, metricRelabelConfig := range endpoint.MetricRelabelConfigs {
			action := metricRelabelConfig.Action
			sourceLabels := metricRelabelConfig.SourceLabels
			if action == "keep" && len(sourceLabels) == 1 && sourceLabels[0] == "__name__" {
				metricsExpressions.Insert(metricRelabelConfig.Regex)
			}
		}
	}
	return metricsExpressions
}
