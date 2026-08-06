package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
)

// suppressEquivalentDurations returns a plan modifier that treats semantically
// equal duration strings ("1m" vs "1m0s" vs "60s") as no change, keeping the
// state value in the plan. Without it, importing a resource whose config spells
// a duration differently than the provider's normalized state form produces a
// permanent phantom diff until the first apply.
func suppressEquivalentDurations() planmodifier.String {
	return durationDiffSuppressor{}
}

type durationDiffSuppressor struct{}

func (durationDiffSuppressor) Description(_ context.Context) string {
	return "Suppresses changes between semantically equal duration strings."
}

func (d durationDiffSuppressor) MarkdownDescription(ctx context.Context) string {
	return d.Description(ctx)
}

func (durationDiffSuppressor) PlanModifyString(_ context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if req.StateValue.IsNull() || req.StateValue.IsUnknown() || req.PlanValue.IsNull() || req.PlanValue.IsUnknown() {
		return
	}
	if monitorV2DurationStringsEqual(req.StateValue.ValueString(), req.PlanValue.ValueString()) {
		resp.PlanValue = req.StateValue
	}
}
