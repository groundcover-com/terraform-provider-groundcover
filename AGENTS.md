# AGENTS.md

Guidance for coding agents (and humans) contributing to `terraform-provider-groundcover`. This file complements [CONTRIBUTING.md](./CONTRIBUTING.md) — it captures the conventions that come up repeatedly in code review.

## Repo layout

- `internal/provider/` — resource implementations (`resource_<name>.go`), API clients (`client_<name>.go`), tests (`resource_<name>_test.go`), shared utils (`yaml_utils.go`, etc.)
- `examples/resources/groundcover_<name>/resource.tf` — HCL example per resource
- `docs/` — generated docs; regenerated from schema + templates via `make generate`
- `templates/` — doc templates consumed by `tfplugindocs`
- `tools/` — doc generation tooling (`cd tools && go generate ./...`)
- `CHANGELOG.md`, `README.md`, `REFERENCE.md` — must be kept in sync with code changes

## PR checklist

Every PR must include this checklist in its description, with items checked as appropriate:

- [ ] I have written acceptance tests for my changes
- [ ] I have run `make generate` to update generated docs
- [ ] I have added examples demonstrating the new functionality
- [ ] I have updated the README.md with details of changes
- [ ] I have updated the CHANGELOG.md file

### PR title convention

Use a clear prefix: `feat:`, `fix:`, `docs:`, `chore:`. When the work is tracked in Linear, prefix the title or branch with the ticket ID (e.g. `BE-1234: ...`).

## Changelog & versioning

Follow semantic versioning (`MAJOR.MINOR.PATCH`):

- **patch** (`x.y.Z+1`) — bug fixes, doc-only changes, internal refactors with no user-facing impact
- **minor** (`x.Y+1.0`) — new resources, new attributes, new optional behavior, SDK bumps that add functionality
- **major** (`X+1.0.0`) — backwards-incompatible changes (removed resources/attributes, breaking schema changes)

Before adding a new version section to `CHANGELOG.md`:

1. Run `git tag --sort=-v:refname | head -5` to see the latest tagged release.
2. Look at `CHANGELOG.md` for any **unreleased** version section above the last tag. If one exists (the version is in the changelog but not tagged yet), **append to that section** — do **not** create a new version above it.
3. Only create a new version section when no unreleased section exists.

## Tests

### Where tests live

Add tests to the **existing test file** for the area you're touching (e.g. add to `yaml_utils_test.go`, not a new `*_extra_test.go`). Don't create parallel test files for new cases.

### Acceptance test scope

Acceptance tests hit a real backend and run slowly (30s–90s+ each). Keep coverage focused on what only acceptance tests can catch:

**Required for every resource:**
- CRUD — create, read, update, delete
- Disappears — resource removed out-of-band still reconciles
- Regressions — e.g. no perpetual plan diff / apply loop

**Optional (one or two, not exhaustive):**
- One major top-level variant of a complex schema (e.g. "Simple" vs "Advanced"). Prefer **unit tests** for finer-grained variant coverage.

**Do not add:**
- One acceptance test per provider in connected apps
- One acceptance test per mutation in synthetic tests
- Exhaustive enum / variant coverage — those belong in unit tests or are already covered by backend tests

If you're reviewing a PR that adds many acceptance test variants beyond CRUD/disappears/regression, suggest moving variant coverage to unit tests.

### Running tests

```bash
# unit only
go test ./internal/provider -v

# all acceptance tests
TF_ACC=1 go test ./internal/provider -v

# single resource
TF_ACC=1 go test ./internal/provider -v -run TestAccPolicyResource
```

Required env vars for acceptance tests:

```bash
export GROUNDCOVER_API_KEY="..."
export GROUNDCOVER_API_URL="https://api.groundcover.com/"
export GROUNDCOVER_BACKEND_ID="..."
# only for ingestion-key tests against non-in-cloud main backend
export GROUNDCOVER_INCLOUD_BACKEND_ID="..."
```

## Adding a new resource

When adding `groundcover_<name>`:

1. `internal/provider/resource_<name>.go` — schema, CRUD, import
2. `internal/provider/client_<name>.go` — API client wrapper
3. `internal/provider/resource_<name>_test.go` — CRUD + disappears + regression acceptance tests
4. `examples/resources/groundcover_<name>/resource.tf` — HCL example
5. `make generate` — refresh `docs/resources/<name>.md`
6. **`README.md`** — add an entry in **both** lists:
   - "Usage Examples" — bullet linking to the example file
   - "Running Tests" — the `TF_ACC=1 go test ... -run TestAcc<Name>Resource` command
7. `CHANGELOG.md` — append to the existing unreleased section, or create one

The README per-resource lists are frequently missed; check both before opening the PR.

## Schema & deprecation

When introducing a new "preferred" config form while keeping the old form working for backward compatibility, it is **intentional** to:

- Accept both forms in the validator
- Document **only** the new form in the schema description and generated docs
- Update examples to use only the new form

Don't "fix" the schema description to also mention the legacy form for consistency with the validator — the omission is deliberate, so new users aren't confused by two ways to do the same thing.

## Common make targets

```bash
make build       # build the provider into ./dist
make install     # go install
make fmt         # gofmt -s -w -e .
make lint        # golangci-lint run
make generate    # regenerate docs (cd tools && go generate ./...)
make test        # unit tests
make testacc     # TF_ACC=1 acceptance tests (long)
```

Run `make fmt lint` and `make generate` before opening a PR.

## Code style

- Follow standard Go idioms; `gofmt` is enforced.
- Match the existing patterns in neighbouring resources rather than introducing new abstractions for a single use case.
- Use lowercase "groundcover" in user-facing strings and docs.
- Handle API errors with clear messages and include retry logic for transient failures where the SDK doesn't already do so.
