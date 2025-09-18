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
  config = jsonencode({
    name = "test-cloudwatch"
    version = 1
    stsRegion = "us-east-1"
    regions = ["us-east-1"]
    roleArn = "arn:aws:iam::123456789012:role/test-role"
    awsMetrics = [
      {
        namespace = "AWS/EC2"
        metrics = [
          {
            name = "CPUUtilization"
            statistics = ["Average"]
            period = 300
            length = 300
            nullAsZero = false
          }
        ]
      }
    ]
    apiConcurrencyLimits = {
      listMetrics = 3
      getMetricData = 5
      getMetricStatistics = 5
      listInventory = 10
    }
    withContextTagsOnInfoMetrics = false
    withInventoryDiscovery = false
  })
  is_paused = false
}

# Output the data integration IDs for reference
output "cloudwatch_dataintegration_id" {
  description = "The ID of the CloudWatch data integration"
  value       = groundcover_dataintegration.cloudwatch_example.id
}