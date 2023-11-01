# `cpv`: Collection Profiles Validator <!--vale off-->

[![Continuous Integration](https://github.com/rexagod/cpv/workflows/ci/badge.svg)](https://github.com/rexagod/cpv/actions) [![Code Quality](https://goreportcard.com/badge/github.com/rexagod/cpv)](https://goreportcard.com/report/github.com/rexagod/cpv) [![API Reference](https://pkg.go.dev/badge/github.com/rexagod/cpv.svg)](https://pkg.go.dev/github.com/rexagod/cpv)

`cpv` is a command line tool for working with [collection profiles](https://github.com/openshift/enhancements/blob/719b231e3b06cf274e77f0d89e46a0d258002572/enhancements/monitoring/scrape-profiles.md?plain=1).

## Usage

`cpv` expects the following set of flags.

<!-- help.md -->

```
Usage of ./cpv:
  -address string
    	Address of the Prometheus instance. (default "http://localhost:9090")
  -allow-list-file string
    	Path to a file containing a list of allow-listed metrics that will always be included within the extracted metrics set. Requires -profile flag to be set.
  -bearer-token string
    	Bearer token for authentication.
  -kubeconfig string
    	Path to kubeconfig file. Defaults to $KUBECONFIG.
  -noisy
    	Enable noisy assumptions: interpret the absence of the collection profiles label as the default 'full' profile (when using the -status flag).
  -output-cardinality
    	Output cardinality of all extracted metrics to a file.
  -profile string
    	Collection profile that the command is being run for.
  -rule-file string
    	Path to a valid rule file to extract metrics from, for eg., https://github.com/prometheus/prometheus/blob/v0.45.0/model/rulefmt/testdata/test.yaml. Requires -profile flag to be set.
  -status
    	Report collection profiles implementation status. -profile may be empty to report status for all profiles.
  -target-selectors string
    	Target selectors used to extract metrics, for eg., https://github.com/prometheus/client_golang/blob/644c80d1360fb1409a3fe8dfc5bad4228f282f3b/api/prometheus/v1/api_test.go#L1007. Requires -profile flag to be set.
  -validate
    	Validate the collection profile implementation. Requires -profile flag to be set.
  -version
    	Print version information.
```

## License

[GNU GPLv3](LICENSE)
