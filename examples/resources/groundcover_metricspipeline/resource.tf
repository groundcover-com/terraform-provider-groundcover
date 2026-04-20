resource "groundcover_metricspipeline" "example" {
  rules = {
    keep_regex = ["http_requests_total", "process_cpu_seconds_total"]
    drop_regex = ["go_.*"]
    add_label = {
      team = "platform"
      env  = "production"
    }
    raw = <<-EOT
      - action: labelmap
        replacement: "groundcover_$1"
    EOT
  }
}
