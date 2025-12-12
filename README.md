# Railway Prometheus Exporter

A custom Prometheus exporter for monitoring your Railway deployments. Export CPU usage, memory consumption, and network metrics from Railway services to Prometheus for simple Railway service monitoring with minimal setup.

## About Railway Prometheus Exporter

The Railway Prometheus Exporter makes it easy to monitor your Railway services using the Prometheus-based technologies you're likely already familiar with.

This exporter uses a pull-based model. When your Prometheus service periodically scrapes the `/metrics` endpoint provided by this service, the exporter fetches the latest metrics from Railway's API.

Metrics are enriched with labels including service name, environment, and project information, making it easy to filter and aggregate data across your Railway infrastructure.

### What is collected currently?

The exporter collects:

- raw metrics:
  - CPU usage
  - memory usage
  - network traffic
- derived metrics:
  - cpu utilization %
  - memory utilization %

## Common Use Cases

- **Resource Monitoring**: Track CPU and memory usage across all Railway services in real-time
- **Cost and Capacity Monitogin**: Monitor resource utilization of your railway services
- **Performance Alerting**: Set up Prometheus alerts when services approach resource limits
- **Multi-Service Observability**: Monitor resource usage across multiple projects and environments with a single service
- **Integration with Grafana**: Visualize Railway metrics alongside more specific application metrics/logs in Grafana dashboards

## Dependencies

- **Go 1.23+**: Comes for free with RailPack
- **Railway API Key**: Account level token makes multi-service monitoring easy
- **Railway Projects**: Active Railway projects and services to monitor
- **Prometheus Server**: To scrape and store the exported metrics
- **Optional: Grafana**: For visualization and dashboarding

## Deployment Dependencies

- [Prometheus Documentation](https://prometheus.io/docs/introduction/overview/)
- [Railway API Documentation](https://docs.railway.com/reference/public-api)

### Optional Dependencies

- [Railway Grafana Stack Template](https://github.com/MykalMachon/railway-grafana-stack) - Deploy Grafana, Prometheus, Loki, and Tempo on Railway with one click
- [Locomotive](https://github.com/FerretCode/locomotive) - Export Railway logs to Loki via webhooks

## Implementation Details

### Quick Start Guide

Just deploy this using the Railway Template!

### Environment Variables

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `RAILWAY_API_KEY` | Your Railway API token | - | Yes |
| `ENVIRONMENT_TARGETS` | Project-environment pairs in format `projectId:environmentId,projectId:environmentId` | - | Yes |
| `PORT` | HTTP server port for metrics endpoint | - | Yes |
| `DEBUG` | Enable debug logging (true/false) | `false` | No |

**Getting Railway Credentials:**

- **API Token**: Navigate to Railway Dashboard -> Settings -> Tokens -> Create new token
- **Project/Environment IDs**: In the Railway dashboard, press `Cmd/Ctrl + K` to open the command palette, then search for "copy project id" or "copy environment id"

**Configuration Examples:**

```bash
# Monitor production environment in multiple projects
ENVIRONMENT_TARGETS=proj-a-id:prod-env-id,proj-b-id:prod-env-id,proj-c-id:prod-env-id

# Monitor different environments per project
ENVIRONMENT_TARGETS=proj-a-id:prod-env-id,proj-b-id:staging-env-id,proj-c-id:dev-env-id

# Monitor multiple environments in a single project
ENVIRONMENT_TARGETS=proj-a-id:prod-env-id,proj-a-id:staging-env-id,proj-a-id:dev-env-id
```

### Available Endpoints

The exporter exposes three HTTP endpoints:

- **`/metrics`** - Prometheus metrics endpoint (scrape target)
- **`/health`** - Health check endpoint (returns "OK")
- **`/`** - landing page with links to metrics and health

### Connecting to Prometheus

Add the exporter as a scrape target in your `prometheus.yml` configuration:

**When deployed on Railway**, use Railway's internal networking:

```yaml
scrape_configs:
  - job_name: 'railway-exporter'
    scrape_interval: 1m
    static_configs:
      - targets: ['railway-prometheus-exporter.railway.internal:9090']
```

### Integration with Grafana Stack

This exporter works seamlessly with another one of my projects, the [Railway Grafana Stack template](https://github.com/MykalMachon/railway-grafana-stack):

1. **Deploy the Grafana Stack** to your Railway project
2. **Deploy this exporter** to the same Railway project
3. **Configure Prometheus** in the Grafana Stack to scrape this exporter:
   - Add the scrape configuration above to Prometheus
   - Use the internal Railway URL for the target
4. **Create Grafana dashboards** using the exported metrics
5. **Set up alerts** in Prometheus/Grafana for resource thresholds

The metrics will automatically appear in Grafana's Explore view and can be used to build custom dashboards for monitoring your Railway services.

**Example Prometheus queries:**

```promql
# Average CPU utilization by service
avg(railway_service_cpu_utilization) by (service_name)

# Memory usage across all services
sum(railway_service_memory_usage_gb) by (project_name, service_name)

# Services approaching CPU limits
railway_service_cpu_utilization > 0.8
```

## Available Metrics

The metrics below are exposed to Prometheus via the `/metrics` endpoint.

### Core Metrics (Directly from the API)

1. `railway_service_cpu_usage`:
   - TYPE: Gauge
   - Units: Cores
   - Description: Current CPU usage of the Railway service.
   - Labels: `environment_id`, `environment_name`, `service_id`, `service_name`, `project_name`
   - Value: `data.metrics[].values[].value` (where `measurement` is `CPU_USAGE`)

2. `railway_service_cpu_limit`:
   - TYPE: Gauge
   - Units: Cores
   - Description: Configured CPU limit for the Railway service.
   - Labels: `environment_id`, `environment_name`, `service_id`, `service_name`, `project_name`
   - Value: `data.metrics[].values[].value` (where `measurement` is `CPU_LIMIT`)

3. `railway_service_memory_usage_gb`:
   - TYPE: Gauge
   - Units: Gigabytes
   - Description: Current memory usage of the Railway service.
   - Labels: `environment_id`, `environment_name`, `service_id`, `service_name`, `project_name`
   - Value: `data.metrics[].values[].value` (where `measurement` is `MEMORY_USAGE_GB`)

4. `railway_service_memory_limit_gb`:
   - TYPE: Gauge
   - Units: Gigabytes
   - Description: Configured memory limit for the Railway service.
   - Labels: `environment_id`, `environment_name`, `service_id`, `service_name`, `project_name`
   - Value: `data.metrics[].values[].value` (where `measurement` is `MEMORY_LIMIT_GB`)

5. `railway_service_network_rx_gb_total`:
   - TYPE: Counter
   - Units: Gigabytes
   - Description: Total network received data for the Railway service.
   - Labels: `environment_id`, `environment_name`, `service_id`, `service_name`, `project_name`
   - Value: `data.metrics[].values[].value` (where `measurement` is `NETWORK_RX_GB`)

6. `railway_service_network_tx_gb_total`:
   - TYPE: Counter
   - Units: Gigabytes
   - Description: Total network transmitted data for the Railway service.
   - Labels: `environment_id`, `environment_name`, `service_id`, `service_name`, `project_name`
   - Value: `data.metrics[].values[].value` (where `measurement` is `NETWORK_TX_GB`)

### Derived Metrics (Calculated in the Exporter)

1. `railway_service_cpu_utilization`:
   - TYPE: Gauge
   - Units: Percentage (0-1)
   - Description: CPU utilization percentage of the Railway service.
   - Labels: `environment_id`, `environment_name`, `service_id`, `service_name`, `project_name`
   - Calculation: `railway_service_cpu_usage / railway_service_cpu_limit`

2. `railway_service_memory_utilization`:
   - TYPE: Gauge
   - Units: Percentage (0-1)
   - Description: Memory utilization percentage of the Railway service.
   - Labels: `environment_id`, `environment_name`, `service_id`, `service_name`, `project_name`
   - Calculation: `railway_service_memory_usage_gb / railway_service_memory_limit_gb`

### Exporter Health Metrics

1. `railway_exporter_last_scrape_success`:
   - TYPE: Gauge
   - Units: Unix timestamp
   - Description: Unix timestamp of the last successful scrape.
   - Labels: None

2. `railway_exporter_last_scrape_duration_seconds`:
   - TYPE: Gauge
   - Units: Seconds
   - Description: Duration of the last scrape.
   - Labels: None

## Inspiration and Similar Projects

- [Locomotive](https://github.com/FerretCode/locomotive): This project was inspired by the work done on Locomotive to allow Railway users to export their Railway logs to Loki and other logging services via webhooks.
- [Railway Grafana Stack](https://github.com/MykalMachon/railway-grafana-stack): another one of my Railway templates that allows you to deploy Grafana, Prometheus, Loki, and Tempo with one click. The Railway Prometheus Exporter works great in conjunction with this template and Locomotive for a complete observability solution.

## Why Deploy Railway Prometheus Exporter on Railway?

Railway is a singular platform to deploy your infrastructure stack. Railway will host your infrastructure so you don't have to deal with configuration, while allowing you to vertically and horizontally scale it.

By deploying Railway Prometheus Exporter on Railway, you are one step closer to supporting a complete full-stack application with minimal burden. Host your servers, databases, AI agents, and more on Railway.
