# Railway Prometheus Exporter

A custom Prometheus exporter for monitoring your Railway deployments. Export CPU usage, memory consumption, and network metrics from Railway services to Prometheus for comprehensive observability and alerting.

## About Railway Prometheus Exporter

The Railway Prometheus Exporter is a specialized monitoring tool that bridges Railway's infrastructure with Prometheus-based observability platforms. It connects to Railway's GraphQL API to collect resource usage metrics from your services and exposes them in Prometheus format for scraping and analysis.

This exporter uses a pull-based collection model where Prometheus periodically scrapes the `/metrics` endpoint, which triggers the exporter to fetch the latest metrics from Railway's API. Metrics are enriched with labels including service name, environment, and project information, making it easy to filter and aggregate data across your Railway infrastructure.

The exporter collects both raw metrics (CPU usage, memory usage, network traffic) and derived metrics (utilization percentages), providing comprehensive visibility into resource consumption and helping you optimize your Railway deployments.

## Common Use Cases

- **Resource Monitoring**: Track CPU and memory usage across all Railway services in real-time
- **Cost Optimization**: Monitor resource utilization to right-size service configurations and reduce costs
- **Performance Alerting**: Set up Prometheus alerts when services approach resource limits
- **Capacity Planning**: Analyze historical resource trends to plan for scaling needs
- **Multi-Service Observability**: Monitor resource usage across multiple projects and environments in a single dashboard
- **Integration with Grafana**: Visualize Railway metrics alongside application metrics in Grafana dashboards

## Dependencies

- **Go 1.23.0+**: Required for building and running the exporter
- **Railway API Key**: Authentication token with permissions to read project metrics
- **Railway Projects**: Active Railway projects and services to monitor
- **Prometheus Server**: To scrape and store the exported metrics
- **Optional: Grafana**: For visualization and dashboarding

## Deployment Dependencies

- [Railway Grafana Stack Template](https://github.com/MykalMachon/railway-grafana-stack) - Deploy Grafana, Prometheus, Loki, and Tempo on Railway
- [Locomotive](https://github.com/FerretCode/locomotive) - Export Railway logs to Loki via webhooks
- [Prometheus Documentation](https://prometheus.io/docs/introduction/overview/)
- [Railway API Documentation](https://docs.railway.com/reference/public-api)

## Implementation Details

### Quick Start Guide

1. **Clone the repository**
   ```bash
   git clone https://github.com/mykalmachon/railway-prometheus-exporter.git
   cd railway-prometheus-exporter
   ```

2. **Configure environment variables**
   ```bash
   cp .env.example .env
   # Edit .env with your Railway credentials
   ```

3. **Set required environment variables**
   - Get your Railway API key from Railway Dashboard → Settings → Tokens
   - Get Project/Environment IDs using `Cmd/Ctrl + K` in Railway dashboard, then search "copy project id" or "copy environment id"

4. **Run locally**
   ```bash
   go run main.go
   ```

5. **Access metrics**
   - Open http://localhost:9090/metrics to view Prometheus metrics
   - Visit http://localhost:9090/health for health status

### Environment Variables

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `RAILWAY_API_KEY` | Your Railway API authentication token | - | Yes |
| `ENVIRONMENT_TARGETS` | Project-environment pairs in format `projectId:environmentId,projectId:environmentId` | - | Yes |
| `PORT` | HTTP server port for metrics endpoint | `9090` | No |
| `DEBUG` | Enable debug logging (true/false) | `false` | No |

**Getting Railway Credentials:**

- **API Token**: Navigate to Railway Dashboard → Settings → Tokens → Create new token
- **Project/Environment IDs**: In the Railway dashboard, press `Cmd/Ctrl + K` to open the command palette, then search for "copy project id" or "copy environment id"

**Configuration Examples:**

```bash
# Monitor production environment in multiple projects
ENVIRONMENT_TARGETS=proj-a:prod-env-id,proj-b:prod-env-id,proj-c:prod-env-id

# Monitor different environments per project
ENVIRONMENT_TARGETS=proj-a:prod-env,proj-b:staging-env,proj-c:dev-env

# Monitor multiple environments in a single project
ENVIRONMENT_TARGETS=proj-a:prod-env,proj-a:staging-env,proj-a:dev-env
```

### Available Endpoints

The exporter exposes three HTTP endpoints:

- **`/metrics`** - Prometheus metrics endpoint (scrape target)
- **`/health`** - Health check endpoint (returns "OK")
- **`/`** - HTML landing page with links to metrics and health

### Connecting to Prometheus

Add the exporter as a scrape target in your `prometheus.yml` configuration:

```yaml
scrape_configs:
  - job_name: 'railway-exporter'
    scrape_interval: 30s
    static_configs:
      - targets: ['localhost:9090']  # For local development
```

**When deployed on Railway**, use Railway's internal networking:

```yaml
scrape_configs:
  - job_name: 'railway-exporter'
    scrape_interval: 30s
    static_configs:
      - targets: ['railway-prometheus-exporter.railway.internal:9090']
```

### Integration with Grafana Stack

This exporter works seamlessly with the [Railway Grafana Stack template](https://github.com/MykalMachon/railway-grafana-stack):

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

### Building and Deployment

**Local Development:**
```bash
# Run directly
go run main.go

# Build binary
go build -o railway-prometheus-exporter

# Run binary
./railway-prometheus-exporter
```

**Deploy to Railway:**

1. **Connect your repository** to Railway
2. **Add a new service** from your GitHub repository
3. **Configure environment variables** in the Railway dashboard:
   - Add `RAILWAY_API_KEY`, `ENVIRONMENT_ID`, and `PROJECT_IDS`
4. **Deploy** - Railway will automatically detect Go and build using NIXPACKS
5. **Health checks** are configured automatically via `railway.json`

The exporter uses Railway's NIXPACKS builder which automatically detects Go projects and handles the build process. Health checks run against the `/health` endpoint with a 30-second timeout.

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

## Why Deploy Railway Prometheus Exporter on Railway?

Railway provides an ideal platform for deploying this Prometheus exporter alongside your monitored applications:

- **Minimal Latency**: Keep your monitoring infrastructure close to your applications for faster metrics collection
- **Internal Networking**: Use Railway's internal URLs for secure, low-latency communication between services
- **Automatic Scaling**: Railway handles resource allocation and scaling based on demand
- **Simple Deployment**: Deploy directly from GitHub with automatic builds using NIXPACKS
- **Native Health Checks**: Built-in health monitoring ensures the exporter stays available
- **Complete Observability Stack**: Deploy alongside the Railway Grafana Stack for end-to-end observability
- **Cost Effective**: Only pay for the resources you use with Railway's usage-based pricing

By deploying the Railway Prometheus Exporter on Railway, you create a complete, self-contained monitoring solution that runs alongside your applications with minimal configuration and maintenance overhead.

## Inspiration and Similar Projects

- [Locomotive](https://github.com/FerretCode/locomotive): This project was inspired by the work done on Locomotive to allow Railway users to export their Railway logs to Loki and other logging services via webhooks.
- [Railway Grafana Stack](https://github.com/MykalMachon/railway-grafana-stack): A Railway template that allows you to deploy Grafana, Prometheus, Loki, and Tempo with one click. The Railway Prometheus Exporter works great in conjunction with this template and Locomotive for a complete observability solution.
