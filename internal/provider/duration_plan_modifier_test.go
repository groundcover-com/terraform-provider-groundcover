package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestSuppressEquivalentDurations(t *testing.T) {
	tests := []struct {
		name     string
		state    types.String
		plan     types.String
		wantPlan types.String
	}{
		{
			name:     "imported state 1m equals config 1m0s",
			state:    types.StringValue("1m"),
			plan:     types.StringValue("1m0s"),
			wantPlan: types.StringValue("1m"),
		},
		{
			name:     "imported state 10m equals config 10m0s",
			state:    types.StringValue("10m"),
			plan:     types.StringValue("10m0s"),
			wantPlan: types.StringValue("10m"),
		},
		{
			name:     "seconds spelling equals minutes spelling",
			state:    types.StringValue("1m"),
			plan:     types.StringValue("60s"),
			wantPlan: types.StringValue("1m"),
		},
		{
			name:     "day spelling equals hours spelling",
			state:    types.StringValue("24h"),
			plan:     types.StringValue("1d"),
			wantPlan: types.StringValue("24h"),
		},
		{
			name:     "real change is kept",
			state:    types.StringValue("1m"),
			plan:     types.StringValue("5m"),
			wantPlan: types.StringValue("5m"),
		},
		{
			name:     "null state on create is kept",
			state:    types.StringNull(),
			plan:     types.StringValue("1m0s"),
			wantPlan: types.StringValue("1m0s"),
		},
		{
			name:     "unknown plan is kept",
			state:    types.StringValue("1m"),
			plan:     types.StringUnknown(),
			wantPlan: types.StringUnknown(),
		},
		{
			name:     "unparseable values are kept",
			state:    types.StringValue("bogus"),
			plan:     types.StringValue("1m"),
			wantPlan: types.StringValue("1m"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := planmodifier.StringRequest{
				StateValue: tt.state,
				PlanValue:  tt.plan,
			}
			resp := &planmodifier.StringResponse{PlanValue: tt.plan}
			suppressEquivalentDurations().PlanModifyString(context.Background(), req, resp)
			if !resp.PlanValue.Equal(tt.wantPlan) {
				t.Fatalf("got plan value %v, want %v", resp.PlanValue, tt.wantPlan)
			}
		})
	}
}
