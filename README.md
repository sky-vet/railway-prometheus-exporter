# Railway Prometheus Exporter

A custom prometheus exporter for monitoring your service's deployments on Railway.

This will eventually export the following information about your railway deployments for consumption by prometheus:

- `CPU_USAGE`: the current usage of the CPU 
- `CPU_LIMIT`: the current limit on CPU usage
- `MEMORY_USAGE_GB`: the current memory usage in gigabytes 
- `MEMORY_LIMIT_GB`: the current memory usage limit in gigabytes
- `NETWORK_RX_GB`: the current network incoming (reception) 
- `NETWORK_TX_GB`: the current network outgoing (transmission)

The actual labels that will be exported via Prometheus are still a work in progress. I am currently reading through [Prometheus's best practices guide for custom exporters](https://prometheus.io/docs/instrumenting/writing_exporters/).

## Available metrics 

The metrics below will be what this exporter exposes to Prometheus via the metrics endpoint.

### Core Metrics (Directly from the API):

1.  `railway_service_cpu_usage`:
    *   TYPE: Gauge
    *   Units: Cores
    *   Description: Current CPU usage of the Railway service.
    *   Labels: `environment_id`, `service_id`
    *   Value: `data.metrics[].values[].value` (where `measurement` is `CPU_USAGE`)

2.  `railway_service_cpu_limit`:
    *   TYPE: Gauge
    *   Units: Cores
    *   Description: Configured CPU limit for the Railway service.
    *   Labels: `environment_id`, `service_id`
    *   Value: `data.metrics[].values[].value` (where `measurement` is `CPU_LIMIT`)

3.  `railway_service_memory_usage_gb`:
    *   TYPE: Gauge
    *   Units: Gigabytes
    *   Description: Current memory usage of the Railway service.
    *   Labels: `environment_id`, `service_id`
    *   Value: `data.metrics[].values[].value` (where `measurement` is `MEMORY_USAGE_GB`)

4.  `railway_service_memory_limit_gb`:
    *   TYPE: Gauge
    *   Units: Gigabytes
    *   Description: Configured memory limit for the Railway service.
    *   Labels: `environment_id`, `service_id`
    *   Value: `data.metrics[].values[].value` (where `measurement` is `MEMORY_LIMIT_GB`)

5.  `railway_service_network_rx_gb_total`:
    *   TYPE: Counter
    *   Units: Gigabytes
    *   Description: Total network received data for the Railway service.
    *   Labels: `environment_id`, `service_id`
    *   Value: `data.metrics[].values[].value` (where `measurement` is `NETWORK_RX_GB`)

6.  `railway_service_network_tx_gb_total`:
    *   TYPE: Counter
    *   Units: Gigabytes
    *   Description: Total network transmitted data for the Railway service.
    *   Labels: `environment_id`, `service_id`
    *   Value: `data.metrics[].values[].value` (where `measurement` is `NETWORK_TX_GB`)

### Derived Metrics (Calculated in the Exporter):

1.  `railway_service_cpu_utilization`:
    *   TYPE: Gauge
    *   Units: Percentage (0-1)
    *   Description: CPU utilization percentage of the Railway service.
    *   Labels: `environment_id`, `service_id`
    *   Calculation: `railway_service_cpu_usage / railway_service_cpu_limit`

2.  `railway_service_memory_utilization`:
    *   TYPE: Gauge
    *   Units: Percentage (0-1)
    *   Description: Memory utilization percentage of the Railway service.
    *   Labels: `environment_id`, `service_id`
    *   Calculation: `railway_service_memory_usage_gb / railway_service_memory_limit_gb`

### Exporter Health Metrics:

1.  `railway_exporter_last_scrape_success`:
    *   TYPE: Gauge
    *   Units: Unix timestamp
    *   Description: Unix timestamp of the last successful scrape.
    *   Labels: None

2.  `railway_exporter_last_scrape_duration_seconds`:
    *   TYPE: Gauge
    *   Units: Seconds
    *   Description: Duration of the last scrape.
    *   Labels: None

## Inspiration and similar projects

- [Locomotive](https://github.com/FerretCode/locomotive): This project was inspired by the work done on Locomotive to allow Railway user's to export their Railway logs to Loki and other logging services via webhooks.
- [Railway Grafana Stack](https://github.com/MykalMachon/railway-grafana-stack): I recently put together a railway template that allows you to deploy Grafana, Prometheus, Loki, and Tempo with one click. The Railway Prometheus Exporter will work great in conjunction with this template and locomotive.