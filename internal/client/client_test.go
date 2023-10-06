package client

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/prometheus/prometheus/promql/parser"
	"k8s.io/apimachinery/pkg/util/sets"
)

var (
	address = os.Getenv("PROMETHEUS_ADDRESS")
)

func isUp(address string) error {
	// nolint: gosec,noctx
	response, err := http.Get(address)

	// go vet complains if we do anything before checking for errors.
	if err != nil {
		return fmt.Errorf("could not establish a connection to '%s'", address)
	}
	defer response.Body.Close()

	return nil
}

func TestGetCardinalityForMetric(t *testing.T) {
	t.Parallel()

	if err := isUp(address); err != nil {
		t.Skip(err)
	}

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

	if err := isUp(address); err != nil {
		t.Skip(err)
	}

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
