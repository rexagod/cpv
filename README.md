# `cpv`: Collection Profiles Validator

`cpv` is a command line tool for validating collection profiles, so the users can ensure that all the required metrics
for a particular collection profile are present at any given point of time. This was ideally made for teams within
OpenShift to adopt the [Scrape Profiles] enhancement and help diversify their monitoring footprint.

[Scrape Profiles]: https://github.com/openshift/enhancements/blob/719b231e3b06cf274e77f0d89e46a0d258002572/enhancements/monitoring/scrape-profiles.md?plain=1

## Usage

`cpv` expects the following set of flags.

```bash
$ ./cpv -h     
Usage of ./cpv:
  -address string
        Address of the Prometheus instance. (default "http://localhost:9090")
  -bearer-token string
        Bearer token for authentication.
  -impl-stats
        Report collection profiles implementation status.
  -kubeconfigPath string
        Path to kubeconfig file. (default "$KUBECONFIG")
  -profile string
        Collection profile to run the validation against.
```

## License

[GNU GPLv3](LICENSE)
