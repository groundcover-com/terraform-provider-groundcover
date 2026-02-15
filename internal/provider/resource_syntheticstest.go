package provider

import (
	"context"
	"errors"
	"fmt"

	"github.com/groundcover-com/groundcover-sdk-go/pkg/models"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure resource implements required interfaces
var (
	_ resource.Resource                   = &syntheticTestResource{}
	_ resource.ResourceWithConfigure      = &syntheticTestResource{}
	_ resource.ResourceWithImportState    = &syntheticTestResource{}
	_ resource.ResourceWithValidateConfig = &syntheticTestResource{}
)

func NewSyntheticTestResource() resource.Resource {
	return &syntheticTestResource{}
}

type syntheticTestResource struct {
	client ApiClient
}

// --- Terraform state models ---

type syntheticTestResourceModel struct {
	ID       types.String `tfsdk:"id"`
	Name     types.String `tfsdk:"name"`
	Enabled  types.Bool   `tfsdk:"enabled"`
	Interval types.String `tfsdk:"interval"`
	Version  types.Int64  `tfsdk:"version"`

	HTTPCheck *syntheticHTTPCheckModel  `tfsdk:"http_check"`
	Assertion []syntheticAssertionModel `tfsdk:"assertion"`
	Retry     *syntheticRetryModel      `tfsdk:"retry"`
	Labels    types.Map                 `tfsdk:"labels"`
}

type syntheticHTTPCheckModel struct {
	URL             types.String `tfsdk:"url"`
	Method          types.String `tfsdk:"method"`
	Timeout         types.String `tfsdk:"timeout"`
	Headers         types.Map    `tfsdk:"headers"`
	FollowRedirects types.Bool   `tfsdk:"follow_redirects"`
	AllowInsecure   types.Bool   `tfsdk:"allow_insecure"`

	Body *syntheticHTTPBodyModel `tfsdk:"body"`
	Auth *syntheticHTTPAuthModel `tfsdk:"auth"`
}

type syntheticHTTPBodyModel struct {
	Type    types.String `tfsdk:"type"`
	Content types.String `tfsdk:"content"`
}

type syntheticHTTPAuthModel struct {
	Type     types.String `tfsdk:"type"`
	Username types.String `tfsdk:"username"`
	Password types.String `tfsdk:"password"`
	Token    types.String `tfsdk:"token"`
}

type syntheticAssertionModel struct {
	Source   types.String `tfsdk:"source"`
	Operator types.String `tfsdk:"operator"`
	Target   types.String `tfsdk:"target"`
	Property types.String `tfsdk:"property"`
	Severity types.String `tfsdk:"severity"`
}

type syntheticRetryModel struct {
	Count    types.Int64  `tfsdk:"count"`
	Interval types.String `tfsdk:"interval"`
}

// --- Schema ---

func (r *syntheticTestResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_synthetic_test"
}

func (r *syntheticTestResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a groundcover Synthetic Test. Synthetic tests allow you to proactively monitor your services by running periodic HTTP checks against specified endpoints.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier (UUID) of the synthetic test.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the synthetic test.",
				Required:    true,
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether the synthetic test is enabled. Default: `true`.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"interval": schema.StringAttribute{
				Description: "How often the check runs. Supported values: `15s`, `30s`, `1m`, `5m`, `10m`, `15m`, `30m`, `1h`.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("15s", "30s", "1m", "5m", "10m", "15m", "30m", "1h"),
				},
			},
			"version": schema.Int64Attribute{
				Description: "Configuration schema version. Managed by the provider.",
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"labels": schema.MapAttribute{
				Description: "Extra labels to attach to the synthetic test metrics.",
				Optional:    true,
				ElementType: types.StringType,
			},
		},
		Blocks: map[string]schema.Block{
			"http_check": schema.SingleNestedBlock{
				Description: "HTTP check configuration. Defines the endpoint to monitor. (Required)",
				Attributes: map[string]schema.Attribute{
					"url": schema.StringAttribute{
						Description: "The URL to check (must include http:// or https://).",
						Required:    true,
					},
					"method": schema.StringAttribute{
						Description: "HTTP method. Supported: `GET`, `POST`, `PUT`, `PATCH`, `DELETE`, `HEAD`, `OPTIONS`.",
						Required:    true,
						Validators: []validator.String{
							stringvalidator.OneOf("GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"),
						},
					},
					"timeout": schema.StringAttribute{
						Description: "Request timeout (e.g. `10s`, `30s`).",
						Optional:    true,
						Computed:    true,
						Default:     stringdefault.StaticString("10s"),
					},
					"headers": schema.MapAttribute{
						Description: "HTTP headers to send with the request.",
						Optional:    true,
						ElementType: types.StringType,
					},
					"follow_redirects": schema.BoolAttribute{
						Description: "Whether to follow HTTP redirects.",
						Optional:    true,
					},
					"allow_insecure": schema.BoolAttribute{
						Description: "Whether to allow insecure TLS connections.",
						Optional:    true,
					},
				},
				Blocks: map[string]schema.Block{
					"body": schema.SingleNestedBlock{
						Description: "HTTP request body.",
						Attributes: map[string]schema.Attribute{
							"type": schema.StringAttribute{
								Description: "Body content type: `json`, `text`, or `raw`.",
								Optional:    true,
								Validators: []validator.String{
									stringvalidator.OneOf("json", "text", "raw"),
								},
							},
							"content": schema.StringAttribute{
								Description: "Body content string.",
								Optional:    true,
							},
						},
					},
					"auth": schema.SingleNestedBlock{
						Description: "HTTP authentication. Supports `basic`, `bearer`, or `none`.",
						Attributes: map[string]schema.Attribute{
							"type": schema.StringAttribute{
								Description: "Auth type: `basic`, `bearer`, or `none`.",
								Optional:    true,
								Validators: []validator.String{
									stringvalidator.OneOf("basic", "bearer", "none"),
								},
							},
							"username": schema.StringAttribute{
								Description: "Username for basic auth.",
								Optional:    true,
							},
							"password": schema.StringAttribute{
								Description: "Password for basic auth. Supports `secretRef::store::<id>` references.",
								Optional:    true,
								Sensitive:   true,
							},
							"token": schema.StringAttribute{
								Description: "Token for bearer auth. Supports `secretRef::store::<id>` references.",
								Optional:    true,
								Sensitive:   true,
							},
						},
					},
				},
			},
			"assertion": schema.ListNestedBlock{
				Description: "Assertions to validate the check result.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"source": schema.StringAttribute{
							Description: "What to assert on: `statusCode`, `responseTime`, `responseHeader`, `jsonBody`, `responseBody`.",
							Required:    true,
							Validators: []validator.String{
								stringvalidator.OneOf("statusCode", "responseTime", "responseHeader", "jsonBody", "responseBody"),
							},
						},
						"operator": schema.StringAttribute{
							Description: "Comparison operator: `eq`, `ne`, `gt`, `lt`, `contains`, `exists`, `notExists`, `startsWith`, `endsWith`, `regex`, `oneOf`.",
							Required:    true,
							Validators: []validator.String{
								stringvalidator.OneOf("eq", "ne", "gt", "lt", "contains", "exists", "notExists", "startsWith", "endsWith", "regex", "oneOf"),
							},
						},
						"target": schema.StringAttribute{
							Description: "Expected value to compare against (as string, e.g. `\"200\"` for status code).",
							Optional:    true,
						},
						"property": schema.StringAttribute{
							Description: "Property path for header or JSON body assertions (e.g. `Content-Type` or `data.id`).",
							Optional:    true,
						},
						"severity": schema.StringAttribute{
							Description: "Assertion severity: `critical` (default) or `degraded`.",
							Optional:    true,
							Validators: []validator.String{
								stringvalidator.OneOf("critical", "degraded"),
							},
						},
					},
				},
			},
			"retry": schema.SingleNestedBlock{
				Description: "Retry policy for failed checks.",
				Attributes: map[string]schema.Attribute{
					"count": schema.Int64Attribute{
						Description: "Number of retry attempts.",
						Optional:    true,
					},
					"interval": schema.StringAttribute{
						Description: "Delay between retries (e.g. `1s`, `500ms`).",
						Optional:    true,
					},
				},
			},
		},
	}
}

// ValidateConfig ensures required blocks are present at plan time.
func (r *syntheticTestResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config syntheticTestResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if config.HTTPCheck == nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("http_check"),
			"Missing http_check block",
			"An http_check block is required to define the synthetic test check configuration.",
		)
	}
}

// Configure adds the provider configured client to the resource.
func (r *syntheticTestResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(ApiClient)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected provider.ApiClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}
	r.client = client
}

// --- CRUD ---

func (r *syntheticTestResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan syntheticTestResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating Synthetic Test", map[string]any{"name": plan.Name.ValueString()})

	if plan.HTTPCheck == nil {
		resp.Diagnostics.AddError(
			"Missing http_check",
			"An http_check block is required to define the synthetic test check configuration.",
		)
		return
	}

	sdkReq := toSDKRequest(&plan)

	createdResp, err := r.client.CreateSyntheticTest(ctx, sdkReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Synthetic Test",
			fmt.Sprintf("Could not create Synthetic Test: %s", err.Error()),
		)
		return
	}

	plan.ID = types.StringValue(createdResp.ID)
	plan.Version = types.Int64Value(1)

	tflog.Debug(ctx, fmt.Sprintf("Synthetic Test created with ID: %s", createdResp.ID))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *syntheticTestResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state syntheticTestResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Reading Synthetic Test resource", map[string]any{"id": state.ID.ValueString()})

	sdkResp, err := r.client.GetSyntheticTest(ctx, state.ID.ValueString())
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			tflog.Warn(ctx, "Synthetic Test not found, removing from state")
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error Reading Synthetic Test",
			fmt.Sprintf("Could not read Synthetic Test: %s", err.Error()),
		)
		return
	}

	if sdkResp == nil {
		tflog.Warn(ctx, "Synthetic Test not found, removing from state")
		resp.State.RemoveResource(ctx)
		return
	}

	fromSDKResponse(ctx, sdkResp, &state)

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r *syntheticTestResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan syntheticTestResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Updating Synthetic Test", map[string]any{"id": plan.ID.ValueString()})

	if plan.HTTPCheck == nil {
		resp.Diagnostics.AddError(
			"Missing http_check",
			"An http_check block is required to define the synthetic test check configuration.",
		)
		return
	}

	sdkReq := toSDKRequest(&plan)

	err := r.client.UpdateSyntheticTest(ctx, plan.ID.ValueString(), sdkReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Synthetic Test",
			fmt.Sprintf("Could not update Synthetic Test: %s", err.Error()),
		)
		return
	}

	plan.Version = types.Int64Value(1)

	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
}

func (r *syntheticTestResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state syntheticTestResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Deleting Synthetic Test resource", map[string]any{"id": state.ID.ValueString()})

	err := r.client.DeleteSyntheticTest(ctx, state.ID.ValueString())
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			tflog.Warn(ctx, "Synthetic Test not found during delete, treating as successful")
		} else {
			resp.Diagnostics.AddError(
				"Error Deleting Synthetic Test",
				fmt.Sprintf("Could not delete Synthetic Test: %s", err.Error()),
			)
			return
		}
	}

	tflog.Debug(ctx, fmt.Sprintf("Successfully deleted Synthetic Test resource with ID %s", state.ID.ValueString()))
}

func (r *syntheticTestResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// --- Conversion: Terraform model → SDK request ---

func toSDKRequest(plan *syntheticTestResourceModel) *models.SyntheticTestCreateRequest {
	sdkReq := &models.SyntheticTestCreateRequest{
		Name:     plan.Name.ValueString(),
		Enabled:  plan.Enabled.ValueBool(),
		Interval: plan.Interval.ValueString(),
		Version:  1, // Always use version 1
	}

	// Build CheckConfig
	checkConfig := &models.WorkerRequest{
		ExecutionPolicy: &models.ExecutionPolicy{
			Assertions: []*models.Assertion{},
		},
		Metadata: &models.Metadata{
			SyntheticName: plan.Name.ValueString(),
			Labels:        map[string]string{},
		},
		Tracing: &models.Tracing{},
	}

	// Labels
	if !plan.Labels.IsNull() && !plan.Labels.IsUnknown() {
		labels := make(map[string]string)
		for k, v := range plan.Labels.Elements() {
			if sv, ok := v.(types.String); ok {
				labels[k] = sv.ValueString()
			}
		}
		checkConfig.Metadata.Labels = labels
	}

	// HTTP Check
	if plan.HTTPCheck != nil {
		checkConfig.Kind = "http"
		httpReq := &models.HTTPRequest{
			Kind:   "http",
			URL:    plan.HTTPCheck.URL.ValueString(),
			Method: plan.HTTPCheck.Method.ValueString(),
		}

		if !plan.HTTPCheck.Timeout.IsNull() {
			httpReq.Timeout = plan.HTTPCheck.Timeout.ValueString()
		}

		if !plan.HTTPCheck.FollowRedirects.IsNull() {
			httpReq.FollowRedirects = plan.HTTPCheck.FollowRedirects.ValueBool()
		}

		if !plan.HTTPCheck.AllowInsecure.IsNull() {
			httpReq.AllowInsecure = plan.HTTPCheck.AllowInsecure.ValueBool()
		}

		if !plan.HTTPCheck.Headers.IsNull() && !plan.HTTPCheck.Headers.IsUnknown() {
			headers := make(map[string]string)
			for k, v := range plan.HTTPCheck.Headers.Elements() {
				if sv, ok := v.(types.String); ok {
					headers[k] = sv.ValueString()
				}
			}
			httpReq.Headers = headers
		}

		if plan.HTTPCheck.Body != nil && (!plan.HTTPCheck.Body.Type.IsNull() || !plan.HTTPCheck.Body.Content.IsNull()) {
			bodyType := plan.HTTPCheck.Body.Type.ValueString()
			if bodyType == "" && !plan.HTTPCheck.Body.Content.IsNull() {
				bodyType = "raw" // default to raw if content is set without type
			}
			httpReq.Body = &models.Body{
				Type:    models.HTTPRequestBodyType(bodyType),
				Content: plan.HTTPCheck.Body.Content.ValueString(),
			}
		}

		if plan.HTTPCheck.Auth != nil && !plan.HTTPCheck.Auth.Type.IsNull() {
			httpReq.Auth = &models.Auth{
				Type: models.HTTPRequestAuthType(plan.HTTPCheck.Auth.Type.ValueString()),
			}
			if !plan.HTTPCheck.Auth.Username.IsNull() {
				httpReq.Auth.Username = plan.HTTPCheck.Auth.Username.ValueString()
			}
			if !plan.HTTPCheck.Auth.Password.IsNull() {
				httpReq.Auth.Password = plan.HTTPCheck.Auth.Password.ValueString()
			}
			if !plan.HTTPCheck.Auth.Token.IsNull() {
				httpReq.Auth.Token = plan.HTTPCheck.Auth.Token.ValueString()
			}
		}

		checkConfig.Request = &models.Request{HTTP: httpReq}
	}

	// Assertions
	if len(plan.Assertion) > 0 {
		assertions := make([]*models.Assertion, 0, len(plan.Assertion))
		for _, a := range plan.Assertion {
			assertion := &models.Assertion{
				Source:   models.AssertionSource(a.Source.ValueString()),
				Operator: models.AssertionOperator(a.Operator.ValueString()),
			}
			if !a.Target.IsNull() {
				assertion.Target = a.Target.ValueString()
			}
			if !a.Property.IsNull() {
				assertion.Property = a.Property.ValueString()
			}
			if !a.Severity.IsNull() {
				assertion.Severity = models.AssertionSeverity(a.Severity.ValueString())
			}
			assertions = append(assertions, assertion)
		}
		checkConfig.ExecutionPolicy.Assertions = assertions
	}

	// Retries
	if plan.Retry != nil {
		checkConfig.ExecutionPolicy.Retries = &models.Retries{
			Count:    plan.Retry.Count.ValueInt64(),
			Interval: plan.Retry.Interval.ValueString(),
		}
	}

	sdkReq.CheckConfig = checkConfig
	return sdkReq
}

// --- Conversion: SDK response → Terraform state ---

func fromSDKResponse(ctx context.Context, sdkResp *models.SyntheticTestCreateRequest, state *syntheticTestResourceModel) {
	state.Name = types.StringValue(sdkResp.Name)
	state.Enabled = types.BoolValue(sdkResp.Enabled)
	state.Interval = types.StringValue(sdkResp.Interval)
	state.Version = types.Int64Value(sdkResp.Version)

	if sdkResp.CheckConfig == nil {
		return
	}

	cc := sdkResp.CheckConfig

	// Labels - set from API response or clear if none returned
	if cc.Metadata != nil && len(cc.Metadata.Labels) > 0 {
		labels := make(map[string]string)
		for k, v := range cc.Metadata.Labels {
			labels[k] = v
		}
		labelsMap, diags := types.MapValueFrom(ctx, types.StringType, labels)
		if !diags.HasError() {
			state.Labels = labelsMap
		}
	} else if !state.Labels.IsNull() {
		// API returned no labels but state had labels - clear them
		state.Labels = types.MapNull(types.StringType)
	}

	// HTTP Check
	if cc.Request != nil && cc.Request.HTTP != nil {
		http := cc.Request.HTTP
		httpModel := &syntheticHTTPCheckModel{
			URL:     types.StringValue(http.URL),
			Method:  types.StringValue(http.Method),
			Timeout: types.StringValue(http.Timeout),
		}

		// Headers - set from API or clear to null when API returns none
		if len(http.Headers) > 0 {
			headersMap, diags := types.MapValueFrom(ctx, types.StringType, http.Headers)
			if !diags.HasError() {
				httpModel.Headers = headersMap
			}
		} else {
			httpModel.Headers = types.MapNull(types.StringType)
		}

		// Set bool fields: preserve explicit user values, set from API if true or if user had set them
		if http.FollowRedirects || (state.HTTPCheck != nil && !state.HTTPCheck.FollowRedirects.IsNull()) {
			httpModel.FollowRedirects = types.BoolValue(http.FollowRedirects)
		}
		if http.AllowInsecure || (state.HTTPCheck != nil && !state.HTTPCheck.AllowInsecure.IsNull()) {
			httpModel.AllowInsecure = types.BoolValue(http.AllowInsecure)
		}

		if http.Body != nil && (http.Body.Content != "" || string(http.Body.Type) != "") {
			httpModel.Body = &syntheticHTTPBodyModel{
				Type:    types.StringValue(string(http.Body.Type)),
				Content: types.StringValue(http.Body.Content),
			}
		}

		if http.Auth != nil && string(http.Auth.Type) != "" {
			authModel := &syntheticHTTPAuthModel{
				Type: types.StringValue(string(http.Auth.Type)),
			}
			if http.Auth.Username != "" {
				authModel.Username = types.StringValue(http.Auth.Username)
			}
			// Don't read back password/token from API - they're sensitive and may be resolved secrets
			// Keep existing state values for these
			if state.HTTPCheck != nil && state.HTTPCheck.Auth != nil {
				authModel.Password = state.HTTPCheck.Auth.Password
				authModel.Token = state.HTTPCheck.Auth.Token
			}
			httpModel.Auth = authModel
		}

		state.HTTPCheck = httpModel
	}

	// Assertions
	if cc.ExecutionPolicy != nil && len(cc.ExecutionPolicy.Assertions) > 0 {
		assertions := make([]syntheticAssertionModel, 0, len(cc.ExecutionPolicy.Assertions))
		for _, a := range cc.ExecutionPolicy.Assertions {
			am := syntheticAssertionModel{
				Source:   types.StringValue(string(a.Source)),
				Operator: types.StringValue(string(a.Operator)),
			}
			if a.Target != "" {
				am.Target = types.StringValue(a.Target)
			}
			if a.Property != "" {
				am.Property = types.StringValue(a.Property)
			}
			if string(a.Severity) != "" {
				am.Severity = types.StringValue(string(a.Severity))
			}
			assertions = append(assertions, am)
		}
		state.Assertion = assertions
	} else {
		state.Assertion = nil
	}

	// Retries - clear when API returns none to avoid stale state
	if cc.ExecutionPolicy != nil && cc.ExecutionPolicy.Retries != nil && cc.ExecutionPolicy.Retries.Count > 0 {
		state.Retry = &syntheticRetryModel{
			Count:    types.Int64Value(cc.ExecutionPolicy.Retries.Count),
			Interval: types.StringValue(cc.ExecutionPolicy.Retries.Interval),
		}
	} else {
		state.Retry = nil
	}
}
