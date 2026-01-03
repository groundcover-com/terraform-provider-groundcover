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

# Example: Prometheus Static Targets
resource "groundcover_dataintegration" "prometheus_static_example" {
  type = "prometheusscrape"
  config = jsonencode({
    version        = 1
    enabled        = true
    name           = "prometheus-static-config"
    scheme         = "https"
    metricsPath    = "/metrics"
    scrapeInterval = "30s"
    scrapeTimeout  = "10s"

    staticTargets = [
      "prometheus-target.example.com:9090"
    ]

    exporters = [
      "prometheus"
    ]

    metricsRelabels = {
      # List of regex of metrics to keep. All other metrics will be dropped.
      keepRegex = [
        "[*]cpu[*]"
      ]

      # List of regex of metrics to drop
      dropRegex = [
        "DB1[*]"
      ]
      raw = <<-EOT
# additional relabeling rules that can be applied such as adding a prefix
  - action: labelmap
    replacement: "groundcover_$1"
EOT
    }

    labelSettings = {
      extraLabels = {
        env = "prod"
      }
    }

    authentication = {
      basicAuth = {
        username = "prometheus-user"
        # refer to groundcover_secret to create a secret
        password = "secretRef::store::d1fc037f11f8ce58"
      }
    }
  })
  is_paused = false
}

# Example: Prometheus HTTPs Target Discovery
resource "groundcover_dataintegration" "prometheus_discovery_example" {
  type = "prometheusscrape"

  config = jsonencode({
    version = 1
    enabled = true
    name    = "Target discovery scraping example"

    exporters = [
      "prometheus"
    ]

    # Durations are numeric (nanoseconds), preserved as-is
    scrapeInterval = 30000000000
    scrapeTimeout  = 10000000000

    metricsPath = "/metrics"
    scheme      = "http"

    # provide the host details for discovery
    httpDiscovery = {
      url = "https://cloud.mongodb.com/prometheus/v1.0/groups/example"

      # Please refer to the groundcover_secret doc on how to create a secret
      authentication = { # Authentication for the discovery endpoint
        basicAuth = {
          username = "prom_user_6909b9ab19480f045c1f2eca"
          password = "secretRef::store::5219731e4bc798eb"
        }
      }
    }
    # Relabeling options on discovered targets. keepRegex - drop all targets which don't comply with this rule. dropRegex - drop all targets which comply with this rule.
    targetsRelabels = {
      keepRegex = [
        ".*shard-00-02.*"
      ]
      dropRegex = [
        ".*shard-00-01.*"
      ]
    }

    authentication = { # Authentication for scraping discovered targets
      basicAuth = {
        username = "prom_user_6909b9ab19480f045c1f2eca"
        password = "secretRef::store::15e1b4b9c0ce0a45"
      }
    }

    metricsRelabels = {
      # List of regex of metrics to keep. All other metrics will be dropped.
      keepRegex = [
        "[*]cpu[*]"
      ]
      dropRegex = [
        "DB1[*]"
      ]
      raw = <<-EOT
# additional relabeling rules that can be applied such as adding a prefix
  - action: labelmap
    replacement: "groundcover_$1"
EOT
    }

    labelSettings = {
      extraLabels = {
        env = "prod"
      }
    }
  })

  is_paused = false
}

# Example: MongoDB Atlas
resource "groundcover_dataintegration" "mongodb_atlas_example" {
  type = "mongoatlasscrape"

  config = jsonencode({
    version = 1
    enabled = true
    name    = "MongoDB Atlas example"

    exporters = [
      "prometheus"
    ]

    # Durations are numeric (nanoseconds), preserved as-is
    scrapeInterval = 30000000000
    scrapeTimeout  = 10000000000

    metricsPath = "/metrics"
    scheme      = "http"

    # provide the host details for discovery
    httpDiscovery = {
      url = "https://cloud.mongodb.com/prometheus/v1.0/groups/example/discovery"

      # Please refer to the groundcover_secret doc on how to create a secret
      authentication = { # Authentication for the discovery endpoint
        basicAuth = {
          username = "prom_user_6909b9ab19480f045c1f2eca"
          password = "secretRef::store::5219731e4bc798eb"
        }
      }
    }
    # Relabeling options on discovered targets. keepRegex - drop all targets which don't comply with this rule. dropRegex - drop all targets which comply with this rule.
    targetsRelabels = {
      keepRegex = [
        ".*shard-00-02.*"
      ]
      dropRegex = [
        ".*shard-00-01.*"
      ]
    }

    authentication = { # Authentication for scraping discovered targets
      basicAuth = {
        username = "prom_user_6909b9ab19480f045c1f2eca"
        password = "secretRef::store::15e1b4b9c0ce0a45"
      }
    }

    metricsRelabels = {
      # List of regex of metrics to keep. All other metrics will be dropped.
      keepRegex = []
      dropRegex = [
        "[*]catalogStats[*]"
      ]
      raw = <<-EOT
# additional relabeling rules that can be applied such as adding a prefix
  - action: labelmap
    replacement: "groundcover_$1"
EOT
    }

    labelSettings = {
      extraLabels = {
        env = "prod"
      }
    }
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

output "prometheus_static_dataintegration_id" {
  description = "The ID of the Prometheus Static Targets data integration"
  value       = groundcover_dataintegration.prometheus_static_example.id
}

output "prometheus_discovery_dataintegration_id" {
  description = "The ID of the Prometheus Target Discovery data integration"
  value       = groundcover_dataintegration.prometheus_discovery_example.id
}

output "mongodb_atlas_dataintegration_id" {
  description = "The ID of the MongoDB Atlas data integration"
  value       = groundcover_dataintegration.mongodb_atlas_example.id
}
