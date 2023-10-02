# `cpv`: Collection Profiles Validator

`cpv` is a command line tool for validating collection profiles, checking their implementation status, and guessing the metrics needed to implement a collection profile.

[Scrape Profiles]: https://github.com/openshift/enhancements/blob/719b231e3b06cf274e77f0d89e46a0d258002572/enhancements/monitoring/scrape-profiles.md?plain=1

## Usage

`cpv` expects the following set of flags.

```bash
┌[rexagod@nebuchadnezzar] [/dev/ttys004] [main] 
└[~/repositories/work/cpv]> go run main.go -h                                                                                                                                                                
Usage of /var/folders/lt/fkdznpv57qjfcvm4p2psgcj00000gn/T/go-build2697341217/b001/exe/main:
  -address string
        Address of the Prometheus instance. (default "http://localhost:9090")
  -bearer-token string
        Bearer token for authentication.
  -guess-profile string
        Guess the metrics needed to implement the given collection profile.
  -guess-profile-param string
        Path to rule file, or targets to be used to guess the metrics needed to implement the --guess-profile.
  -kubeconfig string
        Path to kubeconfig file. (default "$KUBECONFIG")
  -profile string
        Collection profile to run the validation against.
  -status
        Report collection profiles implementation status.
```

## License

[GNU GPLv3](LICENSE)
