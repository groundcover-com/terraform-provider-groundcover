package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// The API echoes a configured "", [] or {} back as empty, and reading that as
// null makes every plan show the empty value being re-added (BE-3828). These
// keep the prior value only when it is a known empty; nothing configured still
// maps to null.

func stringOrPrior(value string, prior types.String) types.String {
	if value == "" {
		if !prior.IsNull() && !prior.IsUnknown() && prior.ValueString() == "" {
			return prior
		}
		return types.StringNull()
	}
	return types.StringValue(value)
}

func stringListOrPrior(ctx context.Context, value []string, prior types.List, diags *diag.Diagnostics) types.List {
	if len(value) == 0 {
		if !prior.IsNull() && !prior.IsUnknown() && len(prior.Elements()) == 0 {
			return prior
		}
		return types.ListNull(types.StringType)
	}
	result, d := types.ListValueFrom(ctx, types.StringType, value)
	diags.Append(d...)
	return result
}

func stringMapOrPrior(ctx context.Context, value map[string]string, prior types.Map, diags *diag.Diagnostics) types.Map {
	if len(value) == 0 {
		if !prior.IsNull() && !prior.IsUnknown() && len(prior.Elements()) == 0 {
			return prior
		}
		return types.MapNull(types.StringType)
	}
	result, d := types.MapValueFrom(ctx, types.StringType, value)
	diags.Append(d...)
	return result
}

func stringSliceOrPrior(values []string, prior []types.String) []types.String {
	if len(values) == 0 {
		if prior != nil && len(prior) == 0 {
			return prior
		}
		return nil
	}
	out := make([]types.String, 0, len(values))
	for _, v := range values {
		out = append(out, types.StringValue(v))
	}
	return out
}
