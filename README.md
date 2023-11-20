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
  -quiet
    	Suppress all output, and use $EDITOR for generated manifests.
  -rule-file string
    	Path to a valid rule file to extract metrics from, for eg., https://github.com/prometheus/prometheus/blob/v0.45.0/model/rulefmt/testdata/test.yaml. Requires -profile flag to be set.
  -status
    	Report collection profiles' implementation status. -profile may be empty to report status for all profiles.
  -target-selectors string
    	Target selectors used to extract metrics, for eg., https://github.com/prometheus/client_golang/blob/644c80d1360fb1409a3fe8dfc5bad4228f282f3b/api/prometheus/v1/api_test.go#L1007. Requires -profile flag to be set.
  -validate
    	Validate the collection profile implementation. Requires -profile flag to be set.
  -version
    	Print version information.
```


### Scenarios

While the utility can be used with the various aforementioned flag combinations to fulfill the desired use-case, the following ones may comparatively be more prominent within the general workflow and thus, have been documented in order to get the developers up-and-running with in no time.

#### Extraction

The utility can be used to extract metrics based a set of given parameters that include:
* `-allow-list-file`: Path to a file containing a list of metrics that will always be included within the extracted metrics set, even if they are not present in the Prometheus instance forwarded at `-address`.
* `-rule-file`: Path to a file containing a set of [`RuleGroup`](https://github.com/prometheus/client_golang/blob/v1.17.0/api/prometheus/v1/api.go#L569)s. All metrics used to define `expr`essions within the `rules` will be extracted. For example, [`model/rulefmt/testdata/test.yaml`](https://github.com/prometheus/prometheus/blob/v0.45.0/model/rulefmt/testdata/test.yaml) will result in the extraction of two metrics: `errors_total` and `requests_total`.
* `-target-selectors`: A set of constraints (resembling [`VectorSelector`](https://github.com/prometheus/prometheus/blob/32ee1b15de6220ab975f3dac7eb82131a0b1e95f/promql/parser/ast.go#L126)s) satisfying the `matchTarget` parameter in [`TargetsMetadata`](https://github.com/prometheus/client_golang/blob/0356577e9b46283f8efae268b73ffee773a6feb7/api/prometheus/v1/api.go#L501). For example. `"{job=\"prometheus\", severity=\"critical\"}"` will result in the extraction of all metrics present in the Prometheus instance forwarded at `-address`, that have the `job` label set to `prometheus` and the `severity` label set to `critical`.

All these flags are mutually exclusive and require the `-profile` flag to be set. Once extracted, the metrics are used to generate a [`RelabelConfig`](https://github.com/prometheus-operator/prometheus-operator/blob/pkg/apis/monitoring/v0.66.0/pkg/apis/monitoring/v1/prometheus_types.go#L1267) that [can be dropped into the `ServiceMonitor` or `PodMonitor` resource](https://github.com/openshift/cluster-monitoring-operator/pull/1785/files#diff-2ced247f66ba1c3c56d30d7ae8c78af6a5eb5e561060d5d64f5caa4cd42626b9R15).

```bash
$ ./cpv -profile="$PROFILE" -rule-file="$RULE_FILE" -target-selectors="$TARGET_SELECTORS" -allow-list-file="$ALLOW_LIST_FILE"
```

```yaml
sourcelabels:
    - __name__
separator: ""
targetlabel: ""
regex: (foo|bar|...)
modulus: 0
replacement: ""
action: keep
```

Additionally, `-output-cardinality` may be specified to output the cardinality of all extracted metrics to a file, in order to better assess decisions around keeping or dropping certain metrics within the `ServiceMonitor` or `PodMonitor` resource(s) for a particular profile.

```
METRIC  CARDINALITY
foo     40
bar     10
...
```

#### Status

The utility can be used to evaluate the extent to which a collection profile has been implemented for every default `ServiceMonitor` or `PodMonitor` resource that has [opted-in to Collection Profiles feature](https://github.com/rexagod/cpv/blob/74ff86c9a7f99635b40f991efc6eb14c859bb496/internal/profiles/utils.go#L48). For example, with respect to the [`default` Kube State Metrics `ServiceMonitor`](https://github.com/JoaoBraveCoding/cluster-monitoring-operator/blob/ad0a06d61793336a7d520cb37d48a053b1b233d1/assets/kube-state-metrics/service-monitor.yaml#L9) (notice the explicit opt-in label), the utility, seeing that this has opted-in to the Collection Profiles feature, will check for the presence of all corresponding [`SupportedNonDefaultCollectionProfiles`](https://github.com/rexagod/cpv/blob/373d577560bae10f10769aeeab33781df7d4dc8f/internal/profiles/types.go#L24) for that `ServiceMonitor` and report the status for each of them (whether they exist or not).

For **all** profiles to be "fully implemented" (i.e., when `-status` is used without specifying a particular `-profile=$PROFILE`) all of the default opted-in `ServiceMonitor` or `PodMonitor` resources (i.e., with `monitoring.openshift.io/collection-profile` label set to `full`) must have the same corresponding resources for every such profile. Here, "corresponding resources" mean the `ServiceMonitor` or `PodMonitor` resources that have their `metadata.name` same as their default opted-in `ServiceMonitor` or `PodMonitor` resource counterpart appended by the profile they fulfill, and with the `monitoring.openshift.io/collection-profile` label set to the profile being checked for.

So, for example, for an opted-in default `ServiceMonitor` resource with `metadata.name` as `kube-state-metrics` and `monitoring.openshift.io/collection-profile: full` present within its label set, the corresponding `ServiceMonitor` resources for the, say, `minimal` profile would be `kube-state-metrics-minimal`. The utility will check for the presence of all corresponding resources for every profile with the default resources' `metadata.name` as the base and report the status for each of them.

```bash
$ ./cpv -profile="$PROFILE" -status
```

```
PROFILE   SERVICE MONITOR  POD MONITOR  ERROR
$PROFILE  foo-monitor                   not implemented
$PROFILE                   bar-monitor  not implemented
...
```

Additionally, a `-noisy` flag may be specified to interpret the absence of `monitoring.openshift.io/collection-profile: full` within the default `ServiceMonitor` or `PodMonitor` resources as the default `full` profile. This is useful when the `ServiceMonitor` or `PodMonitor` resources have not been updated to opt-in to the Collection Profiles feature yet.

#### Validation

The utility can be used to validate against any discrepancies that impact the specified `ServiceMonitor` or `Podmonitor` resources. For this purpose, the utility expects the `-profile` flag, i.e, the profile that the validation should run against, and the `-validate` flag to be set. The validation works by reporting the hierarchy of any missing metrics that the specified `-profile` depends on, the absence of which in turn may end up impacting the resources dependent on those metrics.

```bash
$ ./cpv -profile="$PROFILE" -validate
```

```
$PROFILE MONITOR  GROUP  LOCATION                                                    RULE                         QUERY                                                                                                         METRIC                                            ERROR
etcd-minimal      etcd   .../openshift-etcd-operator-etcd-prometheus-rules-....yaml  etcdMemberCommunicationSlow  histogram_quantile(0.99, rate(etcd_network_peer_round_trip_time_seconds_bucket{job=~".*etcd.*"}[5m])) > 0.15  etcd_network_peer_round_trip_time_seconds_bucket  not loaded
...
```

## License

[GNU GPLv3](LICENSE)
