terraform {
  required_providers {
    groundcover = {
      source = "groundcover-com/groundcover"
    }
  }
}

# Configure the groundcover Provider
provider "groundcover" {
  # api_key can be set via the GROUNDCOVER_API_KEY environment variable
  # backend_id can be set via the GROUNDCOVER_BACKEND_ID environment variable
  # api_url can be set via the GROUNDCOVER_API_URL environment variable (optional)
}

# Example: CloudWatch DataIntegration
# For a full list of supported AWS metrics and statistics, visit the official CloudWatch documentation:
# https://docs.aws.amazon.com/AmazonCloudWatch/latest/monitoring/aws-services-cloudwatch-metrics.html
resource "groundcover_dataintegration" "cloudwatch_example" {
  type = "cloudwatch"
  config = jsonencode({
    name      = "test-cloudwatch"
    version   = 1
    stsRegion = "us-east-1"
    regions   = ["us-east-1", "us-east-2"]
    roleArn   = "arn:aws:iam::123456789012:role/test-role"
    awsMetrics = [
      {
        namespace = "AWS/EC2"
        metrics = [
          {
            name       = "CPUUtilization"
            statistics = ["Average"]
          }
        ]
      }
    ]
    labelSettings = {
      extraLabels = { env = "prod" }
    }
    scrapeInterval = 300000000000
    exporters      = ["prometheus"]
  })
  is_paused = false
  tags = {
    environment = "production"
    team        = "infrastructure"
    cloud       = "aws"
  }
}

# Example: GCP Metrics DataIntegration
# For a full list of supported GCP metrics, visit the official GCP documentation:
# https://cloud.google.com/monitoring/api/metrics_gcp
resource "groundcover_dataintegration" "gcp_example" {
  type = "gcpmetrics"
  config = jsonencode({
    name                 = "test-gcp"
    version              = 1
    enabled              = true
    targetServiceAccount = "demo@gcp-demo-project.iam.gserviceaccount.com"
    regions              = ["us-west1"]
    metricPrefixes       = ["cloudsql.googleapis.com", "compute.googleapis.com"]
    projectIDs           = ["gcp-demo-project"]
    scrapeInterval       = 300000000000
    exporters            = ["prometheus"]
    labelSettings = {
      extraLabels = { env = "prod" }
    }
  })
  is_paused = false
  tags = {
    environment = "production"
    team        = "infrastructure"
    cloud       = "gcp"
  }
}

# Example: Azure Metrics DataIntegration
# For a full list of supported Azure metrics, visit the official Azure Monitor documentation:
# https://learn.microsoft.com/en-us/azure/azure-monitor/reference/metrics-index
resource "groundcover_dataintegration" "azure_example" {
  type = "azuremetrics"
  config = jsonencode({
    name          = "Azure demo"
    version       = 1
    subscriptions = ["b3128f7e-54df-4d2e-9c3e-93a4f1f8c9a0"]
    regions       = ["australiaeast"]
    azureMetrics = [
      {
        resourceType = "Microsoft.Compute/virtualMachines"
        metrics = [
          { name = "Available Memory Bytes" }
        ]
        aggregations = ["Average", "Maximum", "Minimum"]
      }
    ]
    azureCloudEnvironment = "AzurePublicCloud"
    scrapeInterval        = 300000000000
    labelSettings = {
      extraLabels = { env = "prod" }
    }
    exporters = ["prometheus"]
  })
  is_paused = false
  tags = {
    environment = "production"
    team        = "infrastructure"
    cloud       = "azure"
  }
}

# Example: Prometheus Scrape DataIntegration
# For scraping metrics from Prometheus-compatible endpoints
resource "groundcover_dataintegration" "prometheus_example" {
  type = "prometheusscrape"
  config = jsonencode({
    name    = "test-prometheus"
    version = 1
    enabled = true
    staticTargets = [
      "prometheus.example.com:9090"
    ]
    scheme         = "http"
    metricsPath    = "/metrics"
    scrapeInterval = "30s"
    scrapeTimeout  = "10s"
    exporters      = ["prometheus"]
  })
  is_paused = false
  tags = {
    environment = "production"
    team        = "platform"
    type        = "prometheus"
  }
}

# Example: Prometheus Scrape DataIntegration with Basic Auth
# For scraping metrics from authenticated Prometheus-compatible endpoints
variable "prometheus_password" {
  type        = string
  description = "Password for Prometheus basic auth"
  sensitive   = true
}

resource "groundcover_secret" "prometheus_password" {
  name    = "prometheus-basic-auth-password"
  type    = "password"
  content = var.prometheus_password
}

resource "groundcover_dataintegration" "prometheus_with_auth_example" {
  type = "prometheusscrape"
  config = jsonencode({
    name    = "test-prometheus-auth"
    version = 1
    enabled = true
    staticTargets = [
      "prometheus-secure.example.com:9090"
    ]
    scheme      = "https"
    metricsPath = "/metrics"
    authentication = {
      basicAuth = {
        username = "prometheus-user"
        password = groundcover_secret.prometheus_password.id
      }
    }
    scrapeInterval = "30s"
    scrapeTimeout  = "10s"
    exporters      = ["prometheus"]
  })
  is_paused = false
  tags = {
    environment = "production"
    team        = "platform"
    type        = "prometheus"
    auth        = "basic"
  }
}

# Output the data integration IDs for reference
output "cloudwatch_dataintegration_id" {
  description = "The ID of the CloudWatch data integration"
  value       = groundcover_dataintegration.cloudwatch_example.id
}

output "gcpmetrics_dataintegration_id" {
  description = "The ID of the GCP Metrics data integration"
  value       = groundcover_dataintegration.gcp_example.id
}

output "azuremetrics_dataintegration_id" {
  description = "The ID of the Azure Metrics data integration"
  value       = groundcover_dataintegration.azure_example.id
}

output "prometheus_dataintegration_id" {
  description = "The ID of the Prometheus scrape data integration"
  value       = groundcover_dataintegration.prometheus_example.id
}

output "prometheus_with_auth_dataintegration_id" {
  description = "The ID of the Prometheus scrape data integration with basic auth"
  value       = groundcover_dataintegration.prometheus_with_auth_example.id
}

output "prometheus_password_secret_id" {
  description = "The ID of the Prometheus password secret"
  value       = groundcover_secret.prometheus_password.id
}
