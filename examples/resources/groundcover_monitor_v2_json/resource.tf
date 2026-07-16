# groundcover_monitor_v2_json is identical to groundcover_monitor_v2, except
# notification_settings.connected_app_params is a JSON string (via jsonencode) instead of an
# HCL nested map. Prefer it when the config is generated/templated or consumed by tooling that
# can't model nested maps (e.g. the Crossplane provider). For hand-written HCL,
# groundcover_monitor_v2 is usually nicer.
resource "groundcover_monitor_v2_json" "gcql_logs" {
  title            = "GCQL Logs Error Count"
  severity         = "critical"
  measurement_type = "event"

  query {
    type             = "gcql"
    data_type        = "logs"
    expression       = "level:error | stats count() count_all_result"
    instant_rollup   = "5m"
    evaluation_delay = "15m"
  }

  threshold {
    name       = "threshold_1"
    input_name = "threshold_input_query"
    operator   = "gt"
    values     = [10]
  }

  evaluation_interval {
    interval    = "1m"
    pending_for = "5m"
  }

  notification_settings {
    method         = "connectedApps"
    connected_apps = ["slack-connected-app-id"]
    status_filters = ["Alerting", "Resolved"]
    connected_app_params = jsonencode({
      "slack-connected-app-id" = {
        channels = [{ id = "C0123456789", name = "#alerts" }]
      }
    })
  }
}
