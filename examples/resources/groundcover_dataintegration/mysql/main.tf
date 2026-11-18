terraform {
  required_providers {
    groundcover = {
      source = "groundcover-com/groundcover"
    }
  }
}

provider "groundcover" {}

# Replace the cluster, instance endpoints and secret references before applying.
# Create the referenced Kubernetes Secrets in the cluster that runs the integration.
# Use one stable endpoint per instance, including each replica.
# For grants and the optional monitoring-account exclusion, follow the MySQL setup guide.
# Reconnect the monitoring sessions after changing actor rules and reapply the rules after restart.
# https://docs.groundcover.com/collect-data/data-sources/mysql-database-monitoring
locals {
  cluster = "production-cluster"
}

# Primary: measure statements. The server certificate must match the host.
resource "groundcover_dataintegration" "mysql_primary" {
  type    = "mysqldbm"
  cluster = local.cluster

  config = jsonencode({
    version = 1
    name    = "mysql-primary"
    enabled = true
    host    = "mysql-primary.example.com"
    port    = 3306
    secure  = true
    tls = {
      ca = "secretRef::k8s::groundcover::mysql-primary-tls::ca.crt"
    }
    authentication = {
      basicAuth = {
        username = "groundcover"
        password = "secretRef::k8s::groundcover::mysql-primary-monitor::password"
      }
    }
    interval = "1m"
    statements = {
      enabled       = true
      maxQueries    = 200
      filterInserts = false
    }
    healthMetrics   = { enabled = false }
    ignoreDatabases = ["scratch"]
    labelSettings   = { extraLabels = { environment = "production" } }
  })
}

# Replica: collect health metrics using the collector's trusted CAs.
# Disable groups for which the monitoring account has no grants.
resource "groundcover_dataintegration" "mysql_replica" {
  type    = "mysqldbm"
  cluster = local.cluster

  config = jsonencode({
    version = 1
    name    = "mysql-replica"
    enabled = true
    host    = "mysql-replica.example.com"
    authentication = {
      basicAuth = {
        username = "groundcover"
        password = "secretRef::k8s::groundcover::mysql-replica-monitor::password"
      }
    }
    statements = { enabled = false }
    healthMetrics = {
      enabled = true
      groups = {
        transactions = { enabled = false }
        waits        = { enabled = false }
        tableSize    = { enabled = false }
      }
    }
    labelSettings = { extraLabels = { environment = "production" } }
  })
}
