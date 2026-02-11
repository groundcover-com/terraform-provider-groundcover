terraform {
  required_providers {
    groundcover = {
      source = "groundcover-com/groundcover"
    }
  }
}

provider "groundcover" {
  api_key    = "gcsa_AEAQAAAA_AAAAAAAA_AAAAAAAA_AAAAAAAB"
  backend_id = "erez-backend"
  api_url    = "https://erez.main.groundcover.com"
}

resource "groundcover_synthetic_test" "auth_check" {
  name     = "TF Auth Bearer Check"
  enabled  = true
  interval = "1m"

  http_check {
    url     = "https://erez.main.groundcover.com/"
    method  = "GET"
    timeout = "10s"

    auth {
      type  = "bearer"
      token = "eyJhbGciOiJSUzI1NiIsInR5cCI6ImF0K2p3dCIsImtpZCI6IkI1VllDczlSYlpnN1NrV3dvRmNUcSJ9.eyJodHRwczovL2NsaWVudC5pbmZvL29yZyI6Imdyb3VuZGNvdmVyLmNvbSIsImh0dHBzOi8vY2xpZW50LmluZm8vZW1haWwiOiJlcmV6QGdyb3VuZGNvdmVyLmNvbSIsImh0dHBzOi8vZ3JvdW5kY292ZXIvbWFuYWdlZF9zc28iOmZhbHNlLCJodHRwczovL2dyb3VuZGNvdmVyL3JvbGVzIjpbXSwiaHR0cHM6Ly9ncm91bmRjb3Zlci9pc19wZXJzb25hbCI6ZmFsc2UsImlzcyI6Imh0dHBzOi8vZGV2LWF1dGguZ3JvdW5kY292ZXIuY29tLyIsInN1YiI6Imdvb2dsZS1vYXV0aDJ8MTAxMzM4MDA2NTIwOTUwNjE5NTYwIiwiYXVkIjpbImh0dHBzOi8vZ3JvdW5kY292ZXIiLCJodHRwczovL2Rldi1taHY4ZnR3dC51cy5hdXRoMC5jb20vdXNlcmluZm8iXSwiaWF0IjoxNzcwNzU4NzMxLCJleHAiOjE3NzA4NDUxMjksInNjb3BlIjoib3BlbmlkIHByb2ZpbGUgZW1haWwgYWNjZXNzOnJvdXRlciBhY2Nlc3M6cm9sZXMgb2ZmbGluZV9hY2Nlc3MiLCJqdGkiOiJpaTYyeTh5dFV1YTltUUNXS1dhalIxIiwiY2xpZW50X2lkIjoiN3NDalM2bEczSzhBMFQyOTdZZk01QU5uNGhlY29nckgifQ.nQhlTSl_eazJYb5vt3OZEcMRWUDUJ-v4_1JtPVXx0mNYmxIsD0ezCYaNwUM2zELQhN2QtHxtJR9crAwlrkNi5oIKQw6PYeGHjcWQ5yb2Ia9kuWlAq6Fl3IoybtxhD-iZShRqECMdhQekj6HF6gTBQrW2keGmUp4yhfaKUnKMRrDLLnrT-QMzmM5rLAuR2qGScvuCrsTaJ4rw2qQ_kcjuLSrub-RHJTQ7014GxUmXFLwH-RIMgCgRZFYurE34RTUUArSisjSbO_h8wuE7CcTx7qClp8ixk1xyOQG1gvsyyRicsXX0rPnDt37OAlPV9sYHURNQM--b6wFCWhY1wkVCqA"
    }
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

output "synthetic_test_id" {
  value = groundcover_synthetic_test.auth_check.id
}
