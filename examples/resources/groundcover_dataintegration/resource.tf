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
resource "groundcover_dataintegration" "cloudwatch_example" {
  type = "cloudwatch"
  config = <<EOF
stsRegion: us-east-1
regions:
  - us-east-1
roleArn: arn:aws:iam::123456789012:role/test-role
metrics:
  AWS/EC2:
    namespace: AWS/EC2
    metrics:
      - name: CPUUtilization
        statistics:
          - Average
        period: 300
        length: 300
        nullAsZero: false
apiConcurrencyLimits:
  listMetrics: 1
  getMetricData: 5
  getMetricStatistics: 5
  listInventory: 10
withContextTagsOnInfoMetrics: false
withInventoryDiscovery: false
EOF
  is_paused = false
}

# Example: DataDog DataIntegration (paused)
resource "groundcover_dataintegration" "datadog_example" {
  type = "datadog"
  config = <<EOF
api_key: your-datadog-api-key
app_key: your-datadog-app-key
site: datadoghq.com
tags:
  - "env:production"
  - "team:platform"
EOF
  is_paused = true
}

# Output the data integration IDs for reference
output "cloudwatch_dataintegration_id" {
  description = "The ID of the CloudWatch data integration"
  value       = groundcover_dataintegration.cloudwatch_example.id
}

output "datadog_dataintegration_id" {
  description = "The ID of the DataDog data integration"
  value       = groundcover_dataintegration.datadog_example.id
}

output "cloudwatch_dataintegration_created_by" {
  description = "Who created the CloudWatch data integration"
  value       = groundcover_dataintegration.cloudwatch_example.created_by
}
