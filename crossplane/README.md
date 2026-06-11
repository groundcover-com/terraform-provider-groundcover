# groundcover Crossplane provider (upjet)

This directory generates a [Crossplane](https://crossplane.io) provider from the
groundcover Terraform provider in the parent directory, using
[upjet](https://github.com/crossplane/upjet). It adds **custom observe logic** that
mimics the Terraform provider's drift-fixing so Crossplane does not report perpetual
drift on `connected_app`, `monitor`, and `dashboard`.

> Status: **POC** (BE-2055). The hand-written drift-suppression core and the upjet
> configuration are complete and unit-tested, and the generation pipeline has been run
> end-to-end against the in-repo provider — it generates correct CRD API types, deepcopy
> methods, and controllers for all three resources (`monitor`, `dashboard`,
> `connected_app`). The generated tree is reproducible via `make generate` and is
> gitignored while this is a POC (see "What's proven vs. remaining" below).

## Why custom observe is needed

The Terraform provider suppresses cosmetic drift in its Read/plan layer: it preserves the
user's authored value when the server response is *semantically* equivalent (YAML key
order, duration formatting, server-added fields, redacted sensitive data tracked by a
`data_hash`). All of that lives in [`pkg/normalize`](../pkg/normalize).

Upjet reconstructs Terraform state from a managed resource's `spec.forProvider` on every
reconcile and diffs the **raw** observed state, so that suppression is lost — producing
constant drift and consolidation churn on the backend.

`terraform-plugin-framework` resources expose **no** upjet diff-customization hook
(`TerraformCustomDiff` is SDKv2-only; the only PF hook is
`TerraformPluginFrameworkIsStateEmptyFn`). So instead of configuring the diff, we
decorate the runtime external client.

## How the fix works

[`internal/observe`](./internal/observe) provides a `managed.ExternalConnecter`
decorator. After upjet's `Observe` runs, when a resource exists but is reported out of
date, a per-resource `Strategy` re-checks the difference using the **same**
`pkg/normalize` logic as the Terraform provider. If the difference is purely cosmetic,
it forces `ResourceUpToDate = true`, preventing the no-op update loop. Real drift and
inner errors are always passed through unchanged; a comparison error is logged and
treated as "no suppression" so real drift is never hidden.

- `monitor`, `dashboard` → `YAMLUpToDate` (filter to authored keys, normalize, compare).
- `connected_app` → `HashUpToDate` (the `data_hash` forward-looking baseline contract).

## Layout

```
config/                 upjet provider configuration (hand-written)
  provider.go           builds the config.Provider for generation
  external_name.go      external-name handling for the 3 POC resources
  {monitor,dashboard,connectedapp}/config.go
internal/observe/       custom observe decorator + strategies (hand-written, tested)
cmd/generator/          upjet generation entrypoint (`make generate`)
apis/                   GENERATED CRD API types        (produced by `make generate`)
internal/controller/    GENERATED controllers          (produced by `make generate`)
examples/               GENERATED + hand-written sample managed resources
```

## Generation runbook

Prerequisites: Go, the Terraform CLI, and network access.

```bash
cd crossplane
make schema      # produces config/schema.json from the published groundcover provider
make generate    # runs the upjet pipeline: apis/, internal/controller/, examples/
make build test  # compile everything and run the observe unit tests
```

`make schema` pins to a published provider version (see the variables at the top of the
`Makefile`). To generate against the in-repo provider instead, build it locally and point
`PROVIDER_SCHEMA` at a schema produced from that binary.

## Wiring the decorator into generated controllers

Upjet's generated controller `Setup` builds a PF connector and passes it to
`managed.WithExternalConnecter`. To apply suppression, wrap that connector with
`observe.NewConnector` for the three POC resources. Because the generated `Setup` is
regenerated, keep this in a **non-generated** file per resource (e.g.
`internal/controller/monitor/setup_observe.go`) and register it in place of the generated
`Setup` in the controller list.

```go
// internal/controller/monitor/setup_observe.go
package monitor

import (
	tjcontroller "github.com/crossplane/upjet/pkg/controller"
	xpresource "github.com/crossplane/crossplane-runtime/pkg/resource"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/groundcover-com/terraform-provider-groundcover/crossplane/internal/observe"
	"github.com/groundcover-com/terraform-provider-groundcover/crossplane/apis/monitoring/v1alpha1"
)

// monitorFields extracts the authored vs observed monitor YAML. The exact field path
// (Spec.ForProvider.<x> / Status.AtProvider.<x>) comes from the generated type.
func monitorFields(mg xpresource.Managed) (desired, observed string, ok bool) {
	m, ok := mg.(*v1alpha1.Monitor)
	if !ok {
		return "", "", false
	}
	desired = valueOrEmpty(m.Spec.ForProvider.MonitorYaml)
	observed = valueOrEmpty(m.Status.AtProvider.MonitorYaml)
	return desired, observed, desired != "" && observed != ""
}

func SetupWithObserve(mgr ctrl.Manager, o tjcontroller.Options) error {
	inner := tjcontroller.NewTerraformPluginFrameworkAsyncConnector(
		mgr.GetClient(), o.OperationTrackerStore, o.SetupFn,
		o.Provider.Resources["groundcover_monitor"],
		/* PF async options from generated Setup */
	)
	c := observe.NewConnector(inner, observe.NewYAMLStrategy(monitorFields), o.Logger)

	r := managed.NewReconciler(mgr, /* ...same options as generated Setup... */
		managed.WithExternalConnecter(c),
	)
	_ = r
	return nil // register r with the manager exactly as the generated Setup does
}
```

`connected_app` is identical but uses `observe.NewHashStrategy`, extracting the recorded
`data_hash` from `Spec`/`Status` and the current remote hash from `Status.AtProvider`.

## What's proven vs. remaining

Proven by running `make schema && make generate` against the in-repo provider:

- The pipeline generates correct CRD API types, deepcopy methods, and controllers for
  `monitor`, `dashboard`, and `connected_app`.
- `monitor` exposes `MonitorYaml` in both `Spec.ForProvider` and `Status.AtProvider` —
  the clean YAML-strategy case the wiring example above targets.
- `connected_app` exposes `DataHash` in `Status.AtProvider` — exactly what the hash
  strategy consumes.

Findings worth carrying into productionization:

- **Dynamic types.** upjet's `NewProvider` converts the whole schema to an SDKv2 resource
  map and panics on cty `DynamicPseudoType`. `connected_app.data` is dynamic, so the
  generator coerces dynamic attributes to JSON strings before generation
  (`cmd/generator`). upjet then represents the sensitive `data` as a `DataSecretRef`
  (secret reference) on the spec; the desired-vs-observed hash baseline for the hash
  strategy needs design follow-up since `data` is excluded from the diff (`tf:"-"`).
- **Dashboard body.** The generated `dashboard` type exposes `name/description/team/
  preset/override` but no single YAML body field, so the YAML strategy needs the actual
  content field confirmed (or dashboard may need a different suppression approach than
  monitor).
- **Remaining to compile a provider binary.** The generated `apis/zz_register.go` imports
  the upjet-provider-template base packages (`apis/v1alpha1` ProviderConfig, `apis/v1beta1`
  StoreConfig), plus `cmd/provider` and `internal/clients`. Adding that template base
  (and the per-resource `SetupWithObserve` files above) turns the generated tree into a
  runnable provider. That base scaffolding is the next ticket.

## Tests

```bash
go test ./...   # observe decorator decision matrix + YAML/hash comparison
```

The decision logic is proven by fast unit tests. End-to-end validation (apply a managed
resource against a real cluster and confirm no perpetual drift) is a documented manual /
`uptest` step, not part of CI for the POC.
