# examples/resources/groundcover_dashboard/resource.tf

terraform {
  required_providers {
    groundcover = {
      source  = "registry.terraform.io/groundcover-com/groundcover"
      version = ">= 1.0.0"
    }
  }
}

provider "groundcover" {
  # Configure API key and Backend ID via environment variables
  # export GROUNDCOVER_API_KEY="YOUR_API_KEY"
  # export GROUNDCOVER_BACKEND_ID="YOUR_BACKEND_ID"
  api_key    = var.groundcover_api_key
  backend_id = var.groundcover_backend_id
  # api_url = "..." # Optional: Override default API URL
}

variable "groundcover_api_key" {
  type        = string
  description = "Groundcover API Key"
  sensitive   = true
}

variable "groundcover_backend_id" {
  type        = string
  description = "Groundcover Backend ID"
}

# Example Dashboard: Basic metrics dashboard
resource "groundcover_dashboard" "llm_observability_dashboard" {
  name        = "Terraform Example - LLM Observability Dashboard"
  description = "Example dashboard overviewing OpenAI and Anthropic"

  # Dashboard preset contains the JSON configuration with escaped quotes
  preset      = "{\"widgets\":[{\"id\":\"B\",\"type\":\"widget\",\"name\":\"Total LLM Calls\",\"queries\":[{\"id\":\"A\",\"expr\":\"span_type:openai span_type:anthropic | stats by(span_type) count() count_all_result | sort by (count_all_result desc) | limit 5\",\"dataType\":\"traces\",\"editorMode\":\"builder\"}],\"visualizationConfig\":{\"type\":\"stat\"}},{\"id\":\"D\",\"type\":\"widget\",\"name\":\"LLM Calls Rate\",\"queries\":[{\"id\":\"A\",\"expr\":\"sum(rate(groundcover_resource_total_counter{type=~\\\"openai|anthropic\\\",status_code=\\\"ok\\\"})) by (gen_ai_request_model)\",\"dataType\":\"metrics\",\"editorMode\":\"builder\"}],\"visualizationConfig\":{\"type\":\"time-series\",\"selectedChartType\":\"stackedBar\"}},{\"id\":\"E\",\"type\":\"widget\",\"name\":\"Average LLM Response Time\",\"queries\":[{\"id\":\"A\",\"expr\":\"avg(groundcover_resource_latency_seconds{type=~\\\"openai|anthropic\\\"}) by (type)\",\"dataType\":\"metrics\",\"step\":\"disabled\",\"editorMode\":\"builder\"}],\"visualizationConfig\":{\"type\":\"stat\",\"step\":\"disabled\",\"selectedUnit\":\"Seconds\"}},{\"id\":\"A\",\"type\":\"widget\",\"name\":\"Total LLM Tokens Used\",\"queries\":[{\"id\":\"A\",\"expr\":\"span_type:openai span_type:anthropic | stats by(span_type) sum(gen_ai.response.usage.total_tokens) sum_result | sort by (sum_result desc) | limit 5\",\"dataType\":\"traces\",\"editorMode\":\"builder\"}],\"visualizationConfig\":{\"type\":\"stat\",\"step\":\"disabled\"}},{\"id\":\"C\",\"type\":\"widget\",\"name\":\"AVG Input Tokens Per LLM Call \",\"queries\":[{\"id\":\"A\",\"expr\":\"span_type:openai OR span_type:anthropic | stats by(span_type) avg(gen_ai.response.usage.input_tokens) avg_result | sort by (avg_result desc) | limit 5\",\"dataType\":\"traces\",\"editorMode\":\"builder\"}],\"visualizationConfig\":{\"type\":\"stat\"}},{\"id\":\"F\",\"type\":\"widget\",\"name\":\"AVG Output Tokens Per LLM Call \",\"queries\":[{\"id\":\"A\",\"expr\":\"span_type:openai OR span_type:anthropic | stats by(span_type) avg(gen_ai.response.usage.output_tokens) avg_result | sort by (avg_result desc) | limit 5\",\"dataType\":\"traces\",\"editorMode\":\"builder\"}],\"visualizationConfig\":{\"type\":\"stat\",\"step\":\"disabled\"}},{\"id\":\"G\",\"type\":\"widget\",\"name\":\"Top Used Models\",\"queries\":[{\"id\":\"A\",\"expr\":\"span_type:openai OR span_type:anthropic | stats by(gen_ai.request.model) count() count_all_result | sort by (count_all_result desc) | limit 100\",\"dataType\":\"traces\",\"editorMode\":\"builder\"}],\"visualizationConfig\":{\"type\":\"bar\",\"step\":\"disabled\"}},{\"id\":\"H\",\"type\":\"widget\",\"name\":\"Total LLM Errors \",\"queries\":[{\"id\":\"A\",\"expr\":\"(span_type:openai OR span_type:anthropic) status:error | stats by(span_type) count() count_all_result | sort by (count_all_result desc) | limit 1\",\"dataType\":\"traces\",\"editorMode\":\"builder\"}],\"visualizationConfig\":{\"type\":\"stat\"}},{\"id\":\"I\",\"type\":\"widget\",\"name\":\"AVG TTFT Over Time by Model\",\"queries\":[{\"id\":\"A\",\"expr\":\"avg(groundcover_workload_latency_seconds{gen_ai_system=~\\\"openai|anthropic\\\",quantile=\\\"0.50\\\"}) by (gen_ai_request_model)\",\"dataType\":\"metrics\",\"editorMode\":\"builder\"}],\"visualizationConfig\":{\"type\":\"time-series\",\"selectedChartType\":\"line\",\"selectedUnit\":\"Seconds\"}},{\"id\":\"J\",\"type\":\"widget\",\"name\":\"Avg Output Tokens Per Second by Model\",\"queries\":[{\"id\":\"A\",\"expr\":\"avg(groundcover_gen_ai_response_usage_output_tokens{}) by (gen_ai_request_model)\",\"dataType\":\"metrics\",\"editorMode\":\"builder\"},{\"id\":\"B\",\"expr\":\"avg(groundcover_workload_latency_seconds{quantile=\\\"0.50\\\"}) by (gen_ai_request_model)\",\"dataType\":\"metrics\",\"editorMode\":\"builder\"},{\"id\":\"formula-A\",\"expr\":\"A / B\",\"dataType\":\"metrics-formula\",\"editorMode\":\"builder\"}],\"visualizationConfig\":{\"type\":\"time-series\",\"selectedUnit\":\"Number\"}}],\"layout\":[{\"id\":\"B\",\"x\":0,\"y\":0,\"w\":4,\"h\":6,\"minH\":4},{\"id\":\"D\",\"x\":0,\"y\":24,\"w\":24,\"h\":6,\"minH\":4},{\"id\":\"E\",\"x\":8,\"y\":0,\"w\":8,\"h\":6,\"minH\":4},{\"id\":\"A\",\"x\":16,\"y\":0,\"w\":8,\"h\":6,\"minH\":4},{\"id\":\"C\",\"x\":0,\"y\":18,\"w\":8,\"h\":6,\"minH\":4},{\"id\":\"F\",\"x\":8,\"y\":18,\"w\":8,\"h\":6,\"minH\":4},{\"id\":\"G\",\"x\":16,\"y\":18,\"w\":8,\"h\":6,\"minH\":4},{\"id\":\"H\",\"x\":4,\"y\":0,\"w\":4,\"h\":6,\"minH\":4},{\"id\":\"I\",\"x\":0,\"y\":12,\"w\":24,\"h\":6,\"minH\":4},{\"id\":\"J\",\"x\":0,\"y\":3,\"w\":24,\"h\":6,\"minH\":4}],\"duration\":\"Last 15 minutes\",\"variables\":{},\"spec\":{\"layoutType\":\"ordered\"},\"schemaVersion\":4}"
}

# Example Dashboard: Simple monitoring dashboard
resource "groundcover_dashboard" "simple_dashboard" {
  name        = "Simple Dashboard"
  description = "Simple example of a dashboard"
  preset      = "{\"widgets\":[{\"id\":\"A\",\"type\":\"widget\",\"name\":\"avg(groundcover_node_rt_disk_space_used_percent{})\",\"queries\":[{\"id\":\"A\",\"expr\":\"avg(groundcover_node_rt_disk_space_used_percent{})\",\"dataType\":\"metrics\",\"editorMode\":\"builder\"}],\"visualizationConfig\":{\"type\":\"time-series\",\"selectedUnit\":\"Percentage\"}}],\"layout\":[{\"id\":\"A\",\"x\":0,\"y\":0,\"w\":8,\"h\":6,\"minH\":4}],\"duration\":\"Last 15 minutes\",\"variables\":{},\"spec\":{\"layoutType\":\"ordered\"},\"schemaVersion\":4}"
}

output "metrics_dashboard_id" {
  description = "The UUID of the LLM dashboard"
  value       = groundcover_dashboard.llm_observability_dashboard.id
}

output "metrics_dashboard_status" {
  description = "The status of the LLM dashboard"
  value       = groundcover_dashboard.llm_observability_dashboard.status
}

output "simple_dashboard_owner" {
  description = "The owner of the simple dashboard"
  value       = groundcover_dashboard.simple_dashboard.owner
}