resource "groundcover_monitor_v2" "gcql_logs" {
  title            = "GCQL Logs Error Count"
  severity         = "critical"
  measurement_type = "event"
  is_paused        = true

  display {
    header      = "GCQL Logs Error Count"
    description = "Fires when error logs are observed in the evaluation window."
    context_header_labels = [
      "cluster",
      "namespace",
      "workload",
    ]
  }

  query {
    type           = "gcql"
    data_type      = "logs"
    expression     = "level:error | stats count() count_all_result"
    instant_rollup = "5m"
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
    method                  = "connectedApps"
    connected_apps          = ["slack-connected-app-id"]
    status_filters          = ["Alerting", "Resolved"]
    renotification_interval = "4h"
    disable_renotification  = false
    connected_app_params = {
      "slack-connected-app-id" = {
        channels = [{ id = "C0123456789", name = "#alerts" }]
      }
    }
  }

  execution_error_state = "OK"
  no_data_state         = "OK"
}

resource "groundcover_monitor_v2" "gcql_traces" {
  title            = "GCQL Traces Count"
  severity         = "warning"
  measurement_type = "event"
  is_paused        = true

  display {
    header      = "GCQL Traces Count"
    description = "Counts traces returned by GCQL."
  }

  query {
    type           = "gcql"
    data_type      = "traces"
    expression     = "* | stats count() count_all_result"
    instant_rollup = "5m"
  }

  threshold {
    name       = "threshold_1"
    input_name = "threshold_input_query"
    operator   = "gt"
    values     = [1000]
  }

  evaluation_interval {
    interval    = "1m"
    pending_for = "1m"
  }

  execution_error_state = "OK"
  no_data_state         = "OK"
}

resource "groundcover_monitor_v2" "gcql_events" {
  title            = "GCQL Events Count"
  severity         = "warning"
  measurement_type = "event"
  is_paused        = true

  display {
    header      = "GCQL Events Count"
    description = "Counts events returned by GCQL."
  }

  query {
    type           = "gcql"
    data_type      = "events"
    expression     = "* | stats count() count_all_result"
    instant_rollup = "5m"
  }

  threshold {
    name       = "threshold_1"
    input_name = "threshold_input_query"
    operator   = "gt"
    values     = [1000]
  }

  evaluation_interval {
    interval    = "1m"
    pending_for = "1m"
  }

  execution_error_state = "OK"
  no_data_state         = "OK"
}

resource "groundcover_monitor_v2" "gcql_apm" {
  title            = "GCQL APM Count"
  severity         = "warning"
  measurement_type = "event"
  is_paused        = true

  display {
    header      = "GCQL APM Count"
    description = "Counts APM records returned by GCQL."
  }

  query {
    type           = "gcql"
    data_type      = "apm"
    expression     = "* | stats count() count_all_result"
    instant_rollup = "5m"
  }

  threshold {
    name       = "threshold_1"
    input_name = "threshold_input_query"
    operator   = "gt"
    values     = [1000]
  }

  evaluation_interval {
    interval    = "1m"
    pending_for = "1m"
  }

  execution_error_state = "OK"
  no_data_state         = "OK"
}

resource "groundcover_monitor_v2" "metricsql" {
  title            = "MetricsQL Running Pods"
  severity         = "warning"
  measurement_type = "state"
  is_paused        = true

  display {
    header      = "MetricsQL Running Pods"
    description = "Evaluates a MetricsQL query through the Prometheus datasource."
  }

  query {
    type       = "metricsql"
    expression = "sum(groundcover_kube_pod_container_status_running{})"

    relative_timerange {
      from = "-5m"
      to   = "0m"
    }

    rollup {
      function = "last"
      time     = "5m"
    }
  }

  reducer {
    name       = "last_reducer"
    input_name = "threshold_input_query"
    type       = "last"
  }

  threshold {
    name       = "threshold_1"
    input_name = "last_reducer"
    operator   = "gt"
    values     = [1000]

    custom_resolve_threshold {
      operator = "lt"
      values   = [900]
    }
  }

  evaluation_interval {
    interval    = "1m"
    pending_for = "1m"
  }

  execution_error_state = "OK"
  no_data_state         = "OK"
}

resource "groundcover_monitor_v2" "raw_sql" {
  title            = "Raw SQL Constant Query"
  severity         = "warning"
  measurement_type = "event"
  is_paused        = true

  display {
    header      = "Raw SQL Constant Query"
    description = "Evaluates a raw SQL query through the ClickHouse datasource."
  }

  query {
    type       = "raw_sql"
    query_type = "instant"
    expression = "SELECT 0 AS count_all_result"
  }

  threshold {
    name       = "threshold_1"
    input_name = "threshold_input_query"
    operator   = "gt"
    values     = [0]
  }

  evaluation_interval {
    interval    = "1m"
    pending_for = "1m"
  }

  execution_error_state = "OK"
  no_data_state         = "OK"
}
