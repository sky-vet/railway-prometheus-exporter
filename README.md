# Railway Prometheus Exporter

A custom prometheus exporter for monitoring your service's deployments on Railway.

This will eventually export the following information about your railway deployments for consumption by prometheus:

- `CPU_USAGE`: the current usage of the CPU 
- `CPU_LIMIT`: the current limit on CPU usage
- `MEMORY_USAGE_GB`: the current memory usage in gigabytes 
- `MEMORY_LIMIT_GB`: the current memory usage limit in gigabytes
- `NETWORK_RX_GB`: the current network incoming (reception) 
- `NETWORK_TX_GB`: the current network outgoing (transmission)

The actual labels that will be exported via prometheus are still a work in progress. I am currently reading through [Prometheus's best practices guide for custom exporters](https://prometheus.io/docs/instrumenting/writing_exporters/).

## Inspiration and similar projects

- [Locomotive](https://github.com/FerretCode/locomotive): This project was inspired by the work done on Locomotive to allow Railway user's to export their Railway logs to Loki and other logging services via webhooks.
- [Railway Grafana Stack](https://github.com/MykalMachon/railway-grafana-stack): I recently put together a railway template that allows you to deploy Grafana, Prometheus, Loki, and Tempo with one click. The Railway Prometheus Exporter will work great in conjunction with this template and locomotive.