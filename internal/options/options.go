// Package options contains the options for the command.
package options

import (
	"flag"
	"fmt"
	"net/http"
	"os"

	"k8s.io/klog/v2"

	v "github.com/rexagod/cpv/internal/version"
)

var (
	address           string
	allowListFile     string
	bearerToken       string
	kubeconfigPath    string
	noisy             bool
	outputCardinality bool
	profile           string
	ruleFile          string
	status            bool
	targetSelector    string
	validate          bool
	version           bool
)

func init() {
	flag.StringVar(&address, "address", "http://localhost:9090", "Address of the Prometheus instance.")
	flag.StringVar(&bearerToken, "bearer-token", "", "Bearer token for authentication.")
	flag.StringVar(&kubeconfigPath, "kubeconfig", os.Getenv("KUBECONFIG"), "Path to kubeconfig file. Defaults to $KUBECONFIG.")

	// Independent flags.
	flag.BoolVar(&noisy, "noisy", false, "Enable noisy assumptions: interpret the absence of the collection profiles label as the default 'full' profile (when using the -status flag).")
	flag.BoolVar(&outputCardinality, "output-cardinality", false, "Output cardinality of all extracted metrics to a file.")
	flag.StringVar(&profile, "profile", "", "Collection profile that the command is being run for.")
	flag.BoolVar(&version, "version", false, "Print version information.")

	// Dependent flags.
	flag.StringVar(&allowListFile, "allow-list-file", "", "Path to a file containing a list of allow-listed metrics that will always be included within the extracted metrics set. Requires -profile flag to be set.")
	flag.StringVar(&ruleFile, "rule-file", "", "Path to a valid rule file to extract metrics from, for eg., https://github.com/prometheus/prometheus/blob/v0.45.0/model/rulefmt/testdata/test.yaml. Requires -profile flag to be set.")
	flag.BoolVar(&status, "status", false, "Report collection profiles implementation status. -profile may be empty to report status for all profiles.")
	flag.StringVar(&targetSelector, "target-selectors", "", "Target selectors used to extract metrics, for eg., https://github.com/prometheus/client_golang/blob/644c80d1360fb1409a3fe8dfc5bad4228f282f3b/api/prometheus/v1/api_test.go#L1007. Requires -profile flag to be set.")
	flag.BoolVar(&validate, "validate", false, "Validate the collection profile implementation. Requires -profile flag to be set.")

	flag.Parse()

	// Print version information.
	if version {
		v.Println()
		if flag.NFlag() == 1 {
			os.Exit(0)
		}
	}

	if len(bearerToken) == 0 {
		klog.Fatal("Bearer token must be set")
	}
	if len(address) == 0 {
		klog.Fatal("Address must be set")
	}
	if len(kubeconfigPath) == 0 {
		klog.Fatal("KUBECONFIG must be set")
	}
}

// Options contains the options for the command.
type Options struct {
	Address           string
	AllowListFile     string
	BearerToken       string
	KubeconfigPath    string
	Noisy             bool
	OutputCardinality bool
	Profile           string
	RuleFile          string
	Status            bool
	TargetSelectors   string
	Validate          bool
}

func (o *Options) HasExtractor() bool {
	return o.AllowListFile != "" || o.RuleFile != "" || o.TargetSelectors != ""
}

func (o *Options) IsUp() error {
	// nolint: noctx
	response, err := http.Get(o.Address)
	if err != nil {
		return fmt.Errorf("failed to get response from %s: %w", o.Address, err)
	}
	defer response.Body.Close()

	return nil
}

// NewOptions returns a new Options.
func NewOptions() *Options {
	return &Options{
		Address:           address,
		AllowListFile:     allowListFile,
		BearerToken:       bearerToken,
		KubeconfigPath:    kubeconfigPath,
		Noisy:             noisy,
		OutputCardinality: outputCardinality,
		Profile:           profile,
		RuleFile:          ruleFile,
		Status:            status,
		TargetSelectors:   targetSelector,
		Validate:          validate,
	}
}
