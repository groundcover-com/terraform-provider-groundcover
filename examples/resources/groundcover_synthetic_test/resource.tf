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

# Example: With monitor and notification routing
resource "groundcover_synthetic_test" "monitored_check" {
  name     = "Monitored API Check"
  interval = "1m"

  http_check {
    url     = "https://api.example.com/health"
    method  = "GET"
    timeout = "10s"
  }

  assertion {
    source   = "statusCode"
    operator = "eq"
    target   = "200"
  }

  monitor {
    monitor_name            = "API Health Monitor"
    severity                = "S1"
    issue_summary           = "Synthetic check failed: {{ name }}"
    issue_description       = "The synthetic test {{ name }} is failing. Check the endpoint health."
    no_data_state           = "Alerting"
    execution_error_state   = "Alerting"
    renotification_interval = "1h"

    evaluation_interval {
      interval    = "1m"
      pending_for = "0s"
    }

    enabled_workflows = ["workflow-id-1", "workflow-id-2"]
  }
}

# Example: SSL certificate check
resource "groundcover_synthetic_test" "ssl_check" {
  name     = "SSL Certificate Check"
  interval = "5m"

  ssl_check {
    host = "example.com"
    port = 443
  }

  assertion {
    source   = "ssl"
    operator = "exists"
    target   = "true"
    property = "certificateValid"
  }
}

# Example: SSL check with TLS version requirement
resource "groundcover_synthetic_test" "ssl_tls_check" {
  name     = "TLS Version Check"
  interval = "10m"

  ssl_check {
    host        = "api.example.com"
    port        = 443
    verify      = true
    min_version = "1.2"
    sni         = "api.example.com"
    timeout     = "10s"
  }

  assertion {
    source   = "ssl"
    operator = "exists"
    target   = "true"
    property = "certificateValid"
  }

  assertion {
    source   = "ssl"
    operator = "exists"
    target   = "true"
    property = "chainValid"
  }
}

# Example: Basic TCP port check
resource "groundcover_synthetic_test" "tcp_check" {
  name     = "TCP Port Check"
  interval = "1m"

  tcp_check {
    host = "example.com"
    port = 5432
  }

  assertion {
    source   = "tcp"
    operator = "exists"
    target   = "true"
  }
}

# Example: TCP check with send/receive
resource "groundcover_synthetic_test" "tcp_send_check" {
  name     = "TCP Send/Receive Check"
  interval = "5m"

  tcp_check {
    host              = "example.com"
    port              = 6379
    send              = "PING"
    expect_response   = true
    receive_max_bytes = 1024
    timeout           = "10s"
  }

  assertion {
    source   = "tcp"
    operator = "exists"
    target   = "true"
  }
}

output "http_health_check_id" {
  value = groundcover_synthetic_test.http_health_check.id
}

output "http_post_check_id" {
  value = groundcover_synthetic_test.http_post_check.id
}

output "monitored_check_id" {
  value = groundcover_synthetic_test.monitored_check.id
}

output "ssl_check_id" {
  value = groundcover_synthetic_test.ssl_check.id
}

output "tcp_check_id" {
  value = groundcover_synthetic_test.tcp_check.id
}
