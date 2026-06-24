# examples/resources/groundcover_monitor/resource.tf

terraform {
  required_providers {
    groundcover = {
      source  = "registry.terraform.io/groundcover-com/groundcover"
      version = ">= 0.0.0" # Replace with actual version constraint
    }
  }
}

provider "groundcover" {
  # Configure API key and Org Name via environment variables
  # export TF_VAR_groundcover_api_key="YOUR_API_KEY"
  # export TF_VAR_groundcover_backend_id="YOUR_BACKEND_ID"
  api_key    = var.groundcover_api_key
  backend_id = var.groundcover_backend_id
  # api_url = "..." # Optional: Override default API URL
}

variable "groundcover_api_key" {
  type        = string
  description = "groundcover API Key"
  sensitive   = true
}

variable "groundcover_backend_id" {
  type        = string
  description = "groundcover Backend ID"
}

# NOTE: groundcover_monitor is deprecated. Use groundcover_monitor_v2 instead,
# which provides a typed Terraform schema in place of the raw YAML blob.

# Example Monitor: K8s Pod Crashed using monitor_yaml
# Monitor YAML structure docs:
# https://docs.groundcover.com/use-groundcover/monitors/monitor-yaml-structure
resource "groundcover_monitor" "k8s_pod_crashed" {
  monitor_yaml = <<-YAML
title: K8s Pod Crashed Monitor
display:
  header: K8s Pod Crashed - {{ alert.labels.reason }}
  description: |-
    This Monitor fires when a pod has crashed, leading to potential application instability.
    {% if alert.labels.env %} Environment: {{ alert.labels.env }} {% endif %}
    Cluster: {{ alert.labels.cluster }}
    Namespace: {{ alert.labels.namespace }}
    Workload: {{ alert.labels.workload }}
    Pod Name: {{ alert.labels.pod_name }}
    Container: {{ alert.labels.container }}
    Reason: {{ alert.labels.reason }}
  resourceHeaderLabels: []
  contextHeaderLabels:
    - env
    - cluster
    - namespace
    - workload
    - podName
    - container
    - reason
severity: S2
model:
  queries:
    - name: threshold_input_query
      dataType: events
      expression: type:container_crash | stats by (env, cluster, namespace, workload, podName, container, reason) count() crashes_count | rename podName as pod_name
      instantRollup: 5 minutes
  thresholds:
    - name: threshold_1
      inputName: threshold_input_query
      operator: gt
      values:
        - 0
labels: {}
annotations: {}
executionErrorState: OK
noDataState: OK
evaluationInterval:
  interval: 1m
  pendingFor: 0s
notificationSettings:
  method: notificationRoutes
measurementType: event
  YAML
}

output "monitor_example_id" {
  description = "The ID of the example Monitor created via YAML."
  value       = groundcover_monitor.k8s_pod_crashed.id
}

# Note: Accessing specific fields like 'title' directly is not possible
# when using monitor_yaml, as the structure is opaque to Terraform.
# You would need to parse the YAML output if required.
output "monitor_example_yaml_output" {
  description = "The YAML definition applied to the monitor."
  value       = groundcover_monitor.k8s_pod_crashed.monitor_yaml
}