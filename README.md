# Prometheus DPDK Exporter

[![REUSE status](https://api.reuse.software/badge/github.com/ironcore-dev/prometheus-dpdk-exporter)](https://api.reuse.software/info/github.com/ironcore-dev/prometheus-dpdk-exporter)
[![Go Report Card](https://goreportcard.com/badge/github.com/ironcore-dev/prometheus-dpdk-exporter)](https://goreportcard.com/report/github.com/ironcore-dev/prometheus-dpdk-exporter)
[![GitHub License](https://img.shields.io/static/v1?label=License&message=Apache-2.0&color=blue)](LICENSE)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](https://makeapullrequest.com)

Export Dpservice statistics to Prometheus readable format.

## About this project

The `prometheus-dpdk-exporter` is responsible for monitoring and exposing [Dpservice](https://github.com/ironcore-dev/dpservice) statistics from [DPDK telemetry](https://doc.dpdk.org/guides/howto/telemetry.html). When run, `prometheus-dpdk-exporter` creates a simple web server (on a configurable port), on which statistics can be reached. These statistics are updated in configurable time intervals and can be then visualized in dashboard tools like [Grafana](https://grafana.com/). Currently, it provides a solution to get the number of NAT ports used, the number of Virtual services used and other Interface statistics exported as [Prometheus metrics](https://prometheus.io/docs/instrumenting/exposition_formats/).

## Requirements and Setup

[Dpservice](https://github.com/ironcore-dev/dpservice) needs to be running on the same host to run `prometheus-dpdk-exporter` and to export the statistics `prometheus-dpdk-exporter` needs to have access to the socket with the path specified in variable `metrics.SocketPath` *(/var/run/dpdk/rte/dpdk_telemetry.v2)*.
Also specified port (by default 8080) on which we want to run `prometheus-dpdk-exporter` needs to be available.

## Support, Feedback, Contributing

This project is open to feature requests/suggestions, bug reports etc. via [GitHub issues](https://github.com/SAP/<your-project>/issues). Contribution and feedback are encouraged and always welcome. For more information about how to contribute, the project structure, as well as additional contribution information, see our [Contribution Guidelines](CONTRIBUTING.md).

## Code of Conduct

We as members, contributors, and leaders pledge to make participation in our community a harassment-free experience for everyone. By participating in this project, you agree to abide by its [Code of Conduct](CODE_OF_CONDUCT.md) at all times.

## Licensing

Copyright (20xx-)20xx SAP SE or an SAP affiliate company and <your-project> contributors. Please see our [LICENSE](LICENSE) for copyright and license information. Detailed information including third-party components and their licensing/copyright information is available [via the REUSE tool](https://api.reuse.software/info/github.com/SAP/<your-project>).
