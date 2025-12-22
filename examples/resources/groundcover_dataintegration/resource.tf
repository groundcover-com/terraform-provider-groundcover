terraform {
  required_providers {
    groundcover = {
      source = "groundcover-com/groundcover"
    }
  }
}

# Configure the Groundcover Provider
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
