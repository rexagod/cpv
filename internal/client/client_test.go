package client

import (
	"context"
	"os"
	"testing"

	"github.com/prometheus/prometheus/promql/parser"
	"k8s.io/apimachinery/pkg/util/sets"
)

var (
	address = os.Getenv("PROMETHEUS_ADDRESS")
)

func TestGetCardinalityForMetric(t *testing.T) {
	t.Parallel()

	metric := &parser.VectorSelector{
		Name: "up",
	}
	c := NewClient(context.Background(), address, "")
	if err := c.Init(); err != nil {
		t.Fatal(err)
	}
	if k := c.getCardinalityForMetric(metric); k == 0 {
		t.Logf("Got no cardinality for %s", metric)
		t.Fail()
	} else {
		t.Logf("Got %d cardinality for %s", k, metric)
	}
}

func TestEvaluateCardinalities(t *testing.T) {
	t.Parallel()

	metricSet := sets.Set[string]{}
	metricSet.Insert("foo")
	metricSet.Insert("go_cgo_go_to_c_calls_calls_total")
	metricSet.Insert("go_memstats_gc_cpu_fraction")
	metricSet.Insert("job_controller_job_pods_finished_total")
	metricSet.Insert("kubelet_node_name")
	metricSet.Insert("up")

	c := NewClient(context.Background(), address, "")
	if err := c.Init(); err != nil {
		t.Fatal(err)
	}
	cardinalities := c.EvaluateCardinalities(context.Background(), &metricSet)
	if len(cardinalities) != len(metricSet) {
		t.Logf("Expected %d cardinalities, got %d", len(metricSet), len(cardinalities))
		t.Fail()
	}
	for _, c := range cardinalities {
		t.Logf("%s: %d", c.Metric, c.Value)
	}
}
