resource "groundcover_metricspipeline" "example" {
  rules = {
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
