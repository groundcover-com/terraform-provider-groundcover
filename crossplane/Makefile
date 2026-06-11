# Makefile for the groundcover Crossplane provider (upjet-generated).
#
# This provider is generated from the groundcover Terraform provider that lives in the
# parent directory. See README.md for the full generation runbook and the design notes
# in the BE-2055 spec.

# ====================================================================================
# Terraform provider source (POC: pin to a published groundcover provider version).
# To generate against the in-repo provider instead, build it locally and point
# PROVIDER_SCHEMA at the schema produced by `terraform providers schema -json`.

TERRAFORM_PROVIDER_SOURCE        ?= groundcover-com/groundcover
TERRAFORM_PROVIDER_REPO          ?= https://github.com/groundcover-com/terraform-provider-groundcover
TERRAFORM_PROVIDER_VERSION       ?= 1.14.1
TERRAFORM_PROVIDER_DOWNLOAD_NAME ?= terraform-provider-groundcover

PROVIDER_SCHEMA ?= config/schema.json

GOIMPORTS ?= go run golang.org/x/tools/cmd/goimports@latest

# ====================================================================================
# Targets

.PHONY: help
help: ## Show this help.
	@grep -hE '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "} {printf "  %-18s %s\n", $$1, $$2}'

.PHONY: test
test: ## Run unit tests (observe decorator + strategies).
	go test ./...

.PHONY: build
build: ## Build all non-generated packages.
	go build ./...

.PHONY: schema
schema: ## Produce config/schema.json from the Terraform provider (requires terraform CLI).
	@command -v terraform >/dev/null 2>&1 || { echo "terraform CLI is required for 'make schema'"; exit 1; }
	@echo ">> generating $(PROVIDER_SCHEMA) from $(TERRAFORM_PROVIDER_SOURCE) $(TERRAFORM_PROVIDER_VERSION)"
	@mkdir -p .cache/schema
	@printf 'terraform {\n  required_providers {\n    groundcover = {\n      source  = "%s"\n      version = "%s"\n    }\n  }\n}\n' \
		"$(TERRAFORM_PROVIDER_SOURCE)" "$(TERRAFORM_PROVIDER_VERSION)" > .cache/schema/main.tf
	cd .cache/schema && terraform init -upgrade >/dev/null && terraform providers schema -json > ../../$(PROVIDER_SCHEMA)
	@echo ">> wrote $(PROVIDER_SCHEMA)"

.PHONY: generate
generate: $(PROVIDER_SCHEMA) ## Run the upjet generation pipeline (CRDs, controllers, examples).
	@echo ">> running upjet generation pipeline"
	go run ./cmd/generator
	@echo ">> formatting generated code"
	$(GOIMPORTS) -w ./apis ./internal
	go build ./...

$(PROVIDER_SCHEMA):
	@echo "ERROR: $(PROVIDER_SCHEMA) not found. Run 'make schema' first (needs the terraform CLI)."
	@exit 1
