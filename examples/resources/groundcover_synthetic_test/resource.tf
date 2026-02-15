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

# Example: Simple HTTP health check
resource "groundcover_synthetic_test" "http_health_check" {
  name     = "HTTP Health Check"
  interval = "1m"

  http_check {
    url     = "https://httpbin.org/status/200"
    method  = "GET"
    timeout = "10s"
  }

  assertion {
    source   = "statusCode"
    operator = "eq"
    target   = "200"
  }

  assertion {
    source   = "responseTime"
    operator = "lt"
    target   = "5000"
  }

  retry {
    count    = 2
    interval = "1s"
  }
}

# Example: HTTP POST with body and headers
resource "groundcover_synthetic_test" "http_post_check" {
  name     = "HTTP POST API Check"
  interval = "5m"

  http_check {
    url     = "https://httpbin.org/post"
    method  = "POST"
    timeout = "10s"

    headers = {
      "Content-Type" = "application/json"
    }

    body {
      type    = "json"
      content = "{\"test\": \"data\"}"
    }
  }

  assertion {
    source   = "statusCode"
    operator = "eq"
    target   = "200"
  }
}

# Example: Authenticated HTTP check using secrets
resource "groundcover_synthetic_test" "authenticated_check" {
  name     = "Authenticated API Check"
  interval = "5m"

  http_check {
    url     = "https://api.example.com/health"
    method  = "GET"
    timeout = "10s"

    auth {
      type  = "bearer"
      token = "secretRef::store::your-secret-id"
    }
  }

  assertion {
    source   = "statusCode"
    operator = "eq"
    target   = "200"
  }
}

# Example: With custom labels and severity
resource "groundcover_synthetic_test" "performance_check" {
  name     = "Performance Monitoring"
  interval = "30s"

  labels = {
    env  = "production"
    team = "platform"
  }

  http_check {
    url     = "https://api.example.com/status"
    method  = "GET"
    timeout = "5s"
  }

  assertion {
    source   = "statusCode"
    operator = "eq"
    target   = "200"
    severity = "critical"
  }

  assertion {
    source   = "responseTime"
    operator = "lt"
    target   = "300"
    severity = "degraded"
  }

  retry {
    count    = 3
    interval = "500ms"
  }
}

# Output the synthetic test IDs
output "http_health_check_id" {
  value = groundcover_synthetic_test.http_health_check.id
}

output "http_post_check_id" {
  value = groundcover_synthetic_test.http_post_check.id
}
