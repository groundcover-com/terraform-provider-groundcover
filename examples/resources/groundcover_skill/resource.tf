terraform {
  required_providers {
    groundcover = {
      source = "groundcover-com/groundcover"
    }
  }
}

# Organizational Skills require an admin service account.
resource "groundcover_skill" "incident_response" {
  name         = "incident-response"
  description  = "A repeatable workflow for investigating production incidents."
  when_to_use  = "Use when investigating an active production incident or responding to an alert."
  instructions = <<-EOT
    1. Review the active alerts and identify the affected services.
    2. Correlate recent logs, traces, and deployment changes.
    3. Summarize the evidence, likely cause, and recommended next actions.
  EOT
}

output "skill_id" {
  value = groundcover_skill.incident_response.id
}
