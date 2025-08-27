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

# Example: CloudWatch Integration
resource "groundcover_integration" "cloudwatch_example" {
  type = "cloudwatch"
  value = <<EOF
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
}

# Output the integration IDs for reference
output "cloudwatch_integration_id" {
  description = "The ID of the CloudWatch integration"
  value       = groundcover_integration.cloudwatch_example.id
}