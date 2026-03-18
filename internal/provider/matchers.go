// Copyright groundcover 2024
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/groundcover-com/groundcover-sdk-go/pkg/models"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// matcherObjectAttrTypes defines the attribute types for a silence matcher object.
// Shared by both silence and recurring silence resources.
// silenceMatcherModel is the Terraform model for a silence matcher.
// Shared by both silence and recurring silence resources.
type silenceMatcherModel struct {
	Name       types.String `tfsdk:"name"`
	Value      types.String `tfsdk:"value"`
	IsEqual    types.Bool   `tfsdk:"is_equal"`
	IsContains types.Bool   `tfsdk:"is_contains"`
}

var matcherObjectAttrTypes = map[string]attr.Type{
	"name":        types.StringType,
	"value":       types.StringType,
	"is_equal":    types.BoolType,
	"is_contains": types.BoolType,
}

var matcherObjectType = types.ObjectType{AttrTypes: matcherObjectAttrTypes}

// silenceMatchersFromModel converts a Terraform list of matcher models to SDK Matchers.
// Used by both silence and recurring silence resources.
func silenceMatchersFromModel(ctx context.Context, matchersList types.List) (models.Matchers, error) {
	if matchersList.IsNull() || matchersList.IsUnknown() {
		return nil, nil
	}

	var matcherModels []silenceMatcherModel
	diags := matchersList.ElementsAs(ctx, &matcherModels, false)
	if diags.HasError() {
		return nil, fmt.Errorf("failed to extract matchers from list")
	}

	matchers := make(models.Matchers, 0, len(matcherModels))
	for _, m := range matcherModels {
		isEqual := m.IsEqual.ValueBool()
		isContains := m.IsContains.ValueBool()

		matcher := &models.SilenceMatcher{
			Name:    m.Name.ValueString(),
			Value:   m.Value.ValueString(),
			IsEqual: &isEqual,
			IsRegex: &isContains, // SDK uses IsRegex, provider exposes as is_contains
		}
		matchers = append(matchers, matcher)
	}

	return matchers, nil
}

// silenceMatchersToModel converts SDK Matchers to a Terraform list of matcher models.
// Used by both silence and recurring silence resources.
func silenceMatchersToModel(apiMatchers models.Matchers) (types.List, error) {
	if len(apiMatchers) == 0 {
		return types.ListNull(matcherObjectType), nil
	}

	matcherValues := make([]attr.Value, 0, len(apiMatchers))
	for _, m := range apiMatchers {
		isEqual := true
		if m.IsEqual != nil {
			isEqual = *m.IsEqual
		}
		isContains := false
		if m.IsRegex != nil {
			isContains = *m.IsRegex // SDK uses IsRegex, provider exposes as is_contains
		}

		matcherObj, diags := types.ObjectValue(
			matcherObjectAttrTypes,
			map[string]attr.Value{
				"name":        types.StringValue(m.Name),
				"value":       types.StringValue(m.Value),
				"is_equal":    types.BoolValue(isEqual),
				"is_contains": types.BoolValue(isContains),
			},
		)
		if diags.HasError() {
			return types.ListNull(matcherObjectType), fmt.Errorf("failed to create matcher object")
		}
		matcherValues = append(matcherValues, matcherObj)
	}

	matcherList, diags := types.ListValue(matcherObjectType, matcherValues)
	if diags.HasError() {
		return types.ListNull(matcherObjectType), fmt.Errorf("failed to create matchers list")
	}

	return matcherList, nil
}
