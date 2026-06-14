package provider

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-openapi/strfmt"
	"github.com/groundcover-com/groundcover-sdk-go/pkg/models"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"gopkg.in/yaml.v3"
)

var _ resource.Resource = &monitorV2Resource{}
var _ resource.ResourceWithConfigure = &monitorV2Resource{}
var _ resource.ResourceWithImportState = &monitorV2Resource{}
var _ resource.ResourceWithValidateConfig = &monitorV2Resource{}

const (
	monitorV2QueryTypeGCQL      = "gcql"
	monitorV2QueryTypeMetricsQL = "metricsql"
	monitorV2QueryTypeRawSQL    = "raw_sql"

	monitorV2DefaultQueryName = "threshold_input_query"

	monitorV2QueryTypeAnnotationKey = "_gc_monitor_v2_query_type"

	monitorV2DatasourcePrometheus = "prometheus"
	monitorV2DatasourceMetrics    = "metrics"
	monitorV2DatasourceClickhouse = "clickhouse"

	monitorV2QueryTypeInstant = "instant"

	monitorV2DataTypeAnnotationKey = "_gc_data_type"
)

func NewMonitorV2Resource() resource.Resource {
	return &monitorV2Resource{}
}

type monitorV2Resource struct {
	client ApiClient
}

type monitorV2ResourceModel struct {
	ID                   types.String                        `tfsdk:"id"`
	Title                types.String                        `tfsdk:"title"`
	Severity             types.String                        `tfsdk:"severity"`
	MeasurementType      types.String                        `tfsdk:"measurement_type"`
	ExecutionErrorState  types.String                        `tfsdk:"execution_error_state"`
	NoDataState          types.String                        `tfsdk:"no_data_state"`
	IsPaused             types.Bool                          `tfsdk:"is_paused"`
	AutoResolve          types.Bool                          `tfsdk:"auto_resolve"`
	Category             types.String                        `tfsdk:"category"`
	Team                 types.String                        `tfsdk:"team"`
	Labels               types.Map                           `tfsdk:"labels"`
	Annotations          types.Map                           `tfsdk:"annotations"`
	Routing              types.List                          `tfsdk:"routing"`
	Query                *monitorV2QueryModel                `tfsdk:"query"`
	Reducers             []monitorV2ReducerModel             `tfsdk:"reducer"`
	Thresholds           []monitorV2ThresholdModel           `tfsdk:"threshold"`
	EvaluationInterval   *monitorV2EvaluationIntervalModel   `tfsdk:"evaluation_interval"`
	Display              *monitorV2DisplayModel              `tfsdk:"display"`
	NotificationSettings *monitorV2NotificationSettingsModel `tfsdk:"notification_settings"`
}

type monitorV2QueryModel struct {
	Name              types.String                 `tfsdk:"name"`
	Type              types.String                 `tfsdk:"type"`
	Expression        types.String                 `tfsdk:"expression"`
	DataType          types.String                 `tfsdk:"data_type"`
	DatasourceType    types.String                 `tfsdk:"datasource_type"`
	DatasourceID      types.String                 `tfsdk:"datasource_id"`
	QueryType         types.String                 `tfsdk:"query_type"`
	InstantRollup     types.String                 `tfsdk:"instant_rollup"`
	Rollup            *monitorV2RollupModel        `tfsdk:"rollup"`
	RelativeTimerange *monitorV2RelativeRangeModel `tfsdk:"relative_timerange"`
}

type monitorV2RollupModel struct {
	Function types.String `tfsdk:"function"`
	Time     types.String `tfsdk:"time"`
}

type monitorV2RelativeRangeModel struct {
	From types.String `tfsdk:"from"`
	To   types.String `tfsdk:"to"`
}

type monitorV2ReducerModel struct {
	Name              types.String                 `tfsdk:"name"`
	InputName         types.String                 `tfsdk:"input_name"`
	Type              types.String                 `tfsdk:"type"`
	Expression        types.String                 `tfsdk:"expression"`
	RelativeTimerange *monitorV2RelativeRangeModel `tfsdk:"relative_timerange"`
}

type monitorV2ThresholdModel struct {
	Name                   types.String                          `tfsdk:"name"`
	InputName              types.String                          `tfsdk:"input_name"`
	Operator               types.String                          `tfsdk:"operator"`
	Values                 types.List                            `tfsdk:"values"`
	RelativeTimerange      *monitorV2RelativeRangeModel          `tfsdk:"relative_timerange"`
	CustomResolveThreshold *monitorV2CustomResolveThresholdModel `tfsdk:"custom_resolve_threshold"`
}

type monitorV2CustomResolveThresholdModel struct {
	Operator types.String `tfsdk:"operator"`
	Values   types.List   `tfsdk:"values"`
}

type monitorV2EvaluationIntervalModel struct {
	Interval   types.String `tfsdk:"interval"`
	PendingFor types.String `tfsdk:"pending_for"`
}

type monitorV2DisplayModel struct {
	Header               types.String `tfsdk:"header"`
	Description          types.String `tfsdk:"description"`
	ResourceHeaderLabels types.List   `tfsdk:"resource_header_labels"`
	ContextHeaderLabels  types.List   `tfsdk:"context_header_labels"`
	TemplateLanguage     types.String `tfsdk:"template_language"`
}

type monitorV2NotificationSettingsModel struct {
	Method                 types.String `tfsdk:"method"`
	ConnectedApps          types.List   `tfsdk:"connected_apps"`
	ConnectedAppParams     types.Map    `tfsdk:"connected_app_params"`
	StatusFilters          types.List   `tfsdk:"status_filters"`
	DisableRenotification  types.Bool   `tfsdk:"disable_renotification"`
	RenotificationInterval types.String `tfsdk:"renotification_interval"`
}

type monitorV2ConnectedAppDeliveryOptionsModel struct {
	Channels types.List `tfsdk:"channels"`
}

func (r *monitorV2Resource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_monitor_v2"
}

func (r *monitorV2Resource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a groundcover Monitor with a typed Terraform schema. This resource avoids the raw YAML blob used by `groundcover_monitor` and supports GCQL, MetricsQL, and raw SQL query definitions.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Monitor identifier (UUID).",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"title": schema.StringAttribute{
				MarkdownDescription: "Monitor title.",
				Required:            true,
			},
			"severity": schema.StringAttribute{
				MarkdownDescription: "Monitor severity.",
				Required:            true,
			},
			"measurement_type": schema.StringAttribute{
				MarkdownDescription: "Type of measurement evaluated by the monitor.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.OneOf("state", "event"),
				},
			},
			"execution_error_state": schema.StringAttribute{
				MarkdownDescription: "State to enter if query execution fails.",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.OneOf("OK", "Error", "Alerting"),
				},
			},
			"no_data_state": schema.StringAttribute{
				MarkdownDescription: "State to enter when the query returns no data.",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.OneOf("OK", "NoData", "Alerting"),
				},
			},
			"is_paused": schema.BoolAttribute{
				MarkdownDescription: "Whether the monitor is paused.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"auto_resolve": schema.BoolAttribute{
				MarkdownDescription: "Whether the monitor should auto-resolve.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"category": schema.StringAttribute{
				MarkdownDescription: "Monitor category.",
				Optional:            true,
			},
			"team": schema.StringAttribute{
				MarkdownDescription: "Team associated with the monitor.",
				Optional:            true,
			},
			"labels": schema.MapAttribute{
				MarkdownDescription: "Labels to attach to the monitor and resulting alerts.",
				Optional:            true,
				Computed:            true,
				ElementType:         types.StringType,
			},
			"annotations": schema.MapAttribute{
				MarkdownDescription: fmt.Sprintf("Annotations to attach to the monitor and resulting alerts. The `%s` and `%s` keys are reserved for provider-managed Monitor V2 state and cannot be configured.", monitorV2QueryTypeAnnotationKey, monitorV2DataTypeAnnotationKey),
				Optional:            true,
				Computed:            true,
				ElementType:         types.StringType,
			},
			"routing": schema.ListAttribute{
				MarkdownDescription: "Routing destinations for the monitor.",
				Optional:            true,
				Computed:            true,
				ElementType:         types.StringType,
			},
		},
		Blocks: map[string]schema.Block{
			"query": schema.SingleNestedBlock{
				MarkdownDescription: "Required. Single query evaluated by the monitor. Use `type = \"gcql\"`, `type = \"metricsql\"`, or `type = \"raw_sql\"`.",
				Attributes: map[string]schema.Attribute{
					"name": schema.StringAttribute{
						MarkdownDescription: fmt.Sprintf("Query name. Defaults to `%s`.", monitorV2DefaultQueryName),
						Optional:            true,
						Computed:            true,
					},
					"type": schema.StringAttribute{
						MarkdownDescription: "Query language/type. Supported values: `gcql`, `metricsql`, `raw_sql`.",
						Required:            true,
						Validators: []validator.String{
							stringvalidator.OneOf(monitorV2QueryTypeGCQL, monitorV2QueryTypeMetricsQL, monitorV2QueryTypeRawSQL),
						},
					},
					"expression": schema.StringAttribute{
						MarkdownDescription: "Query expression.",
						Required:            true,
					},
					"data_type": schema.StringAttribute{
						MarkdownDescription: "GCQL data type. Required when `type = \"gcql\"`. Supported values: `logs`, `traces`, `events`, `entities`, `rum`, `issues`.",
						Optional:            true,
						Validators: []validator.String{
							stringvalidator.OneOf("logs", "traces", "events", "entities", "rum", "issues"),
						},
					},
					"datasource_type": schema.StringAttribute{
						MarkdownDescription: "Metrics datasource type for MetricsQL. Defaults to `prometheus` when `type = \"metricsql\"`.",
						Optional:            true,
						Computed:            true,
						Validators: []validator.String{
							stringvalidator.OneOf(monitorV2DatasourcePrometheus, monitorV2DatasourceMetrics, monitorV2DatasourceClickhouse),
						},
					},
					"datasource_id": schema.StringAttribute{
						MarkdownDescription: "Optional datasource identifier for raw queries.",
						Optional:            true,
					},
					"query_type": schema.StringAttribute{
						MarkdownDescription: "Query execution type for MetricsQL or raw SQL. Defaults to `instant`.",
						Optional:            true,
						Computed:            true,
						Validators: []validator.String{
							stringvalidator.OneOf("instant", "range"),
						},
					},
					"instant_rollup": schema.StringAttribute{
						MarkdownDescription: "GCQL rollup window used to add the monitor evaluation time bucket, for example `5m` or `5 minutes`.",
						Optional:            true,
						Computed:            true,
					},
				},
				Blocks: map[string]schema.Block{
					"rollup": schema.SingleNestedBlock{
						MarkdownDescription: "MetricsQL rollup. Required for MetricsQL queries.",
						Attributes: map[string]schema.Attribute{
							"function": schema.StringAttribute{
								MarkdownDescription: "Rollup function.",
								Optional:            true,
								Validators: []validator.String{
									stringvalidator.OneOf("avg", "max", "min", "sum", "count", "stddev", "stdvar", "last"),
								},
							},
							"time": schema.StringAttribute{
								MarkdownDescription: "Rollup time window, for example `5m` or `1 hour`.",
								Optional:            true,
								Computed:            true,
							},
						},
					},
					"relative_timerange": relativeTimerangeBlock(),
				},
			},
			"reducer": schema.ListNestedBlock{
				MarkdownDescription: "Reducers that aggregate or transform query results before threshold evaluation.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							MarkdownDescription: "Reducer output name.",
							Optional:            true,
						},
						"input_name": schema.StringAttribute{
							MarkdownDescription: "Query or reducer output name to use as input.",
							Optional:            true,
						},
						"type": schema.StringAttribute{
							MarkdownDescription: "Reducer type.",
							Required:            true,
							Validators: []validator.String{
								stringvalidator.OneOf("last", "min", "max", "mean", "sum", "count", "math"),
							},
						},
						"expression": schema.StringAttribute{
							MarkdownDescription: "Math expression when `type = \"math\"`.",
							Optional:            true,
						},
					},
					Blocks: map[string]schema.Block{
						"relative_timerange": relativeTimerangeBlock(),
					},
				},
			},
			"threshold": schema.ListNestedBlock{
				MarkdownDescription: "Required. Thresholds that decide when the monitor fires. At least one threshold block must be configured.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							MarkdownDescription: "Threshold name.",
							Required:            true,
						},
						"input_name": schema.StringAttribute{
							MarkdownDescription: "Query or reducer output name to evaluate.",
							Required:            true,
						},
						"operator": schema.StringAttribute{
							MarkdownDescription: "Threshold comparison operator.",
							Required:            true,
							Validators: []validator.String{
								stringvalidator.OneOf("gt", "lt", "gte", "lte", "eq", "neq", "within_range", "outside_range", "within_range_included", "outside_range_included"),
							},
						},
						"values": schema.ListAttribute{
							MarkdownDescription: "Numeric threshold values.",
							Required:            true,
							ElementType:         types.Float64Type,
						},
					},
					Blocks: map[string]schema.Block{
						"relative_timerange":       relativeTimerangeBlock(),
						"custom_resolve_threshold": customResolveThresholdBlock(),
					},
				},
			},
			"evaluation_interval": schema.SingleNestedBlock{
				MarkdownDescription: "Monitor evaluation interval and pending duration.",
				Attributes: map[string]schema.Attribute{
					"interval": schema.StringAttribute{
						MarkdownDescription: "How often the monitor evaluates, for example `1m`.",
						Optional:            true,
						Computed:            true,
					},
					"pending_for": schema.StringAttribute{
						MarkdownDescription: "How long the condition must remain true before alerting, for example `5m`.",
						Optional:            true,
						Computed:            true,
					},
				},
			},
			"display": schema.SingleNestedBlock{
				MarkdownDescription: "Display metadata shown in monitor issues.",
				Attributes: map[string]schema.Attribute{
					"header": schema.StringAttribute{
						MarkdownDescription: "Issue header.",
						Optional:            true,
					},
					"description": schema.StringAttribute{
						MarkdownDescription: "Issue description.",
						Optional:            true,
					},
					"resource_header_labels": schema.ListAttribute{
						MarkdownDescription: "Labels shown in the resource header.",
						Optional:            true,
						ElementType:         types.StringType,
					},
					"context_header_labels": schema.ListAttribute{
						MarkdownDescription: "Labels shown in the context header.",
						Optional:            true,
						ElementType:         types.StringType,
					},
					"template_language": schema.StringAttribute{
						MarkdownDescription: "Template language for header and description.",
						Optional:            true,
						Validators: []validator.String{
							stringvalidator.OneOf("jinja2"),
						},
					},
				},
			},
			"notification_settings": schema.SingleNestedBlock{
				MarkdownDescription: "Notification behavior for this monitor.",
				Attributes: map[string]schema.Attribute{
					"method": schema.StringAttribute{
						MarkdownDescription: "Notification method.",
						Optional:            true,
						Validators: []validator.String{
							stringvalidator.OneOf("notificationRoutes", "connectedApps", "noNotifications"),
						},
					},
					"connected_apps": schema.ListAttribute{
						MarkdownDescription: "Connected app IDs to notify.",
						Optional:            true,
						ElementType:         types.StringType,
					},
					"connected_app_params": schema.MapNestedAttribute{
						MarkdownDescription: "Per-connected-app delivery options keyed by connected app ID.",
						Optional:            true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"channels": schema.ListAttribute{
									MarkdownDescription: "Slack channel IDs to notify for this connected app.",
									Optional:            true,
									ElementType:         types.StringType,
								},
							},
						},
					},
					"status_filters": schema.ListAttribute{
						MarkdownDescription: "Issue statuses that should notify.",
						Optional:            true,
						ElementType:         types.StringType,
						Validators: []validator.List{
							listvalidator.ValueStringsAre(stringvalidator.OneOf("Alerting", "Resolved")),
						},
					},
					"disable_renotification": schema.BoolAttribute{
						MarkdownDescription: "Whether renotification is disabled.",
						Optional:            true,
						Computed:            true,
						Default:             booldefault.StaticBool(false),
					},
					"renotification_interval": schema.StringAttribute{
						MarkdownDescription: "Duration between renotifications, for example `4h`.",
						Optional:            true,
					},
				},
			},
		},
	}
}

func customResolveThresholdBlock() schema.SingleNestedBlock {
	return schema.SingleNestedBlock{
		MarkdownDescription: "Optional custom recovery threshold used to reduce alert flapping. Supported only with `gt`, `lt`, `within_range`, and `outside_range` firing operators.",
		Attributes: map[string]schema.Attribute{
			"operator": schema.StringAttribute{
				MarkdownDescription: "Recovery comparison operator.",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.OneOf("gt", "lt", "within_range", "outside_range"),
				},
			},
			"values": schema.ListAttribute{
				MarkdownDescription: "Numeric recovery threshold values.",
				Optional:            true,
				ElementType:         types.Float64Type,
			},
		},
	}
}

func relativeTimerangeBlock() schema.SingleNestedBlock {
	return schema.SingleNestedBlock{
		MarkdownDescription: "Relative time range for this query, reducer, or threshold.",
		Attributes: map[string]schema.Attribute{
			"from": schema.StringAttribute{
				MarkdownDescription: "Start of the relative range, for example `-5m`.",
				Optional:            true,
				Computed:            true,
			},
			"to": schema.StringAttribute{
				MarkdownDescription: "End of the relative range, for example `0m`.",
				Optional:            true,
				Computed:            true,
			},
		},
	}
}

func (r *monitorV2Resource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *monitorV2Resource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config monitorV2ResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	monitorV2ValidateAnnotations(config.Annotations, &resp.Diagnostics)

	if config.Query == nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("query"),
			"Missing query block",
			"groundcover_monitor_v2 requires exactly one query block.",
		)
		return
	}

	queryType := monitorV2String(config.Query.Type)
	if queryType == "" {
		return
	}

	switch queryType {
	case monitorV2QueryTypeGCQL:
		monitorV2ValidateUnsupportedQueryFields(config.Query, queryType, &resp.Diagnostics)
		if monitorV2String(config.Query.DataType) == "" {
			resp.Diagnostics.AddAttributeError(
				path.Root("query").AtName("data_type"),
				"Missing GCQL data type",
				"`data_type` is required when query.type is `gcql`.",
			)
		}
	case monitorV2QueryTypeMetricsQL:
		monitorV2ValidateUnsupportedQueryFields(config.Query, queryType, &resp.Diagnostics)
		if config.Query.Rollup == nil {
			resp.Diagnostics.AddAttributeError(
				path.Root("query").AtName("rollup"),
				"Missing MetricsQL rollup",
				"`rollup` is required when query.type is `metricsql`.",
			)
		} else {
			if monitorV2String(config.Query.Rollup.Function) == "" {
				resp.Diagnostics.AddAttributeError(
					path.Root("query").AtName("rollup").AtName("function"),
					"Missing MetricsQL rollup function",
					"`rollup.function` is required when query.type is `metricsql`.",
				)
			}
			if monitorV2String(config.Query.Rollup.Time) == "" {
				resp.Diagnostics.AddAttributeError(
					path.Root("query").AtName("rollup").AtName("time"),
					"Missing MetricsQL rollup time",
					"`rollup.time` is required when query.type is `metricsql`.",
				)
			}
		}
	case monitorV2QueryTypeRawSQL:
		monitorV2ValidateUnsupportedQueryFields(config.Query, queryType, &resp.Diagnostics)
		if config.Query.Rollup != nil {
			resp.Diagnostics.AddAttributeError(
				path.Root("query").AtName("rollup"),
				"Unsupported raw SQL rollup",
				"`rollup` is only supported for MetricsQL queries.",
			)
		}
	}

	if len(config.Thresholds) == 0 {
		resp.Diagnostics.AddAttributeError(
			path.Root("threshold"),
			"Missing threshold block",
			"groundcover_monitor_v2 requires at least one threshold block.",
		)
	}

	for i, threshold := range config.Thresholds {
		if threshold.CustomResolveThreshold == nil {
			continue
		}
		parentOp := monitorV2String(threshold.Operator)
		if parentOp != "" && !monitorV2SupportsCustomResolveOperator(parentOp) {
			resp.Diagnostics.AddAttributeError(
				path.Root("threshold").AtListIndex(i).AtName("operator"),
				"Unsupported threshold operator for custom resolve threshold",
				"`custom_resolve_threshold` is supported only when threshold.operator is one of `gt`, `lt`, `within_range`, or `outside_range`.",
			)
		}
		if !threshold.CustomResolveThreshold.Operator.IsUnknown() && monitorV2String(threshold.CustomResolveThreshold.Operator) == "" {
			resp.Diagnostics.AddAttributeError(
				path.Root("threshold").AtListIndex(i).AtName("custom_resolve_threshold").AtName("operator"),
				"Missing custom resolve threshold operator",
				"`custom_resolve_threshold.operator` is required when custom_resolve_threshold is configured.",
			)
		}
		if !threshold.CustomResolveThreshold.Values.IsUnknown() && len(monitorV2Float64List(ctx, threshold.CustomResolveThreshold.Values, &resp.Diagnostics)) == 0 {
			resp.Diagnostics.AddAttributeError(
				path.Root("threshold").AtListIndex(i).AtName("custom_resolve_threshold").AtName("values"),
				"Missing custom resolve threshold values",
				"`custom_resolve_threshold.values` is required when custom_resolve_threshold is configured.",
			)
		}
	}

	monitorV2ValidateNotificationSettings(config.NotificationSettings, &resp.Diagnostics)
}

func monitorV2ValidateUnsupportedQueryFields(query *monitorV2QueryModel, queryType string, diags *diag.Diagnostics) {
	if query == nil {
		return
	}

	switch queryType {
	case monitorV2QueryTypeGCQL:
		monitorV2RejectConfiguredQueryString(query.DatasourceType, "datasource_type", queryType, diags)
		monitorV2RejectConfiguredQueryString(query.DatasourceID, "datasource_id", queryType, diags)
		monitorV2RejectConfiguredQueryString(query.QueryType, "query_type", queryType, diags)
		if query.Rollup != nil {
			monitorV2RejectQueryBlock("rollup", queryType, diags)
		}
	case monitorV2QueryTypeMetricsQL:
		monitorV2RejectConfiguredQueryString(query.DataType, "data_type", queryType, diags)
		monitorV2RejectConfiguredQueryString(query.DatasourceID, "datasource_id", queryType, diags)
		monitorV2RejectConfiguredQueryString(query.InstantRollup, "instant_rollup", queryType, diags)
	case monitorV2QueryTypeRawSQL:
		monitorV2RejectConfiguredQueryString(query.DataType, "data_type", queryType, diags)
		monitorV2RejectConfiguredQueryString(query.DatasourceType, "datasource_type", queryType, diags)
		monitorV2RejectConfiguredQueryString(query.InstantRollup, "instant_rollup", queryType, diags)
	}
}

func monitorV2RejectConfiguredQueryString(value types.String, name, queryType string, diags *diag.Diagnostics) {
	if value.IsNull() || value.IsUnknown() || value.ValueString() == "" {
		return
	}

	diags.AddAttributeError(
		path.Root("query").AtName(name),
		"Unsupported query field",
		fmt.Sprintf("`query.%s` is not supported when query.type is `%s`.", name, queryType),
	)
}

func monitorV2RejectQueryBlock(name, queryType string, diags *diag.Diagnostics) {
	diags.AddAttributeError(
		path.Root("query").AtName(name),
		"Unsupported query block",
		fmt.Sprintf("`query.%s` is not supported when query.type is `%s`.", name, queryType),
	)
}

func monitorV2ValidateAnnotations(annotations types.Map, diags *diag.Diagnostics) {
	if annotations.IsNull() || annotations.IsUnknown() {
		return
	}

	for key := range annotations.Elements() {
		if !monitorV2IsInternalAnnotation(key) {
			continue
		}
		diags.AddAttributeError(
			path.Root("annotations").AtMapKey(key),
			"Reserved monitor annotation",
			fmt.Sprintf("`annotations.%s` is reserved for provider-managed Monitor V2 state and cannot be configured.", key),
		)
	}
}

func monitorV2ValidateNotificationSettings(settings *monitorV2NotificationSettingsModel, diags *diag.Diagnostics) {
	if settings == nil {
		return
	}

	methodUnknown := settings.Method.IsUnknown()
	methodSet := !settings.Method.IsNull() && !methodUnknown
	isConnectedApps := methodSet && settings.Method.ValueString() == "connectedApps"

	hasAppsSet := !settings.ConnectedApps.IsNull() && !settings.ConnectedApps.IsUnknown()
	hasAppsNonEmpty := hasAppsSet && len(settings.ConnectedApps.Elements()) > 0
	hasParamsSet := !settings.ConnectedAppParams.IsNull() && !settings.ConnectedAppParams.IsUnknown()
	hasFiltersSet := !settings.StatusFilters.IsNull() && !settings.StatusFilters.IsUnknown()

	if methodUnknown {
		return
	}

	if isConnectedApps && !settings.ConnectedApps.IsUnknown() && !hasAppsNonEmpty {
		diags.AddAttributeError(
			path.Root("notification_settings").AtName("connected_apps"),
			"Missing connected apps",
			"`notification_settings.connected_apps` must be set and non-empty when notification_settings.method is `connectedApps`.",
		)
	}
	if !isConnectedApps && hasAppsSet {
		diags.AddAttributeError(
			path.Root("notification_settings").AtName("connected_apps"),
			"Invalid notification settings combination",
			"`notification_settings.connected_apps` can only be set when notification_settings.method is `connectedApps`.",
		)
	}
	if !isConnectedApps && hasParamsSet {
		diags.AddAttributeError(
			path.Root("notification_settings").AtName("connected_app_params"),
			"Invalid notification settings combination",
			"`notification_settings.connected_app_params` can only be set when notification_settings.method is `connectedApps`.",
		)
	}
	if !isConnectedApps && hasFiltersSet {
		diags.AddAttributeError(
			path.Root("notification_settings").AtName("status_filters"),
			"Invalid notification settings combination",
			"`notification_settings.status_filters` can only be set when notification_settings.method is `connectedApps`.",
		)
	}
}

func (r *monitorV2Resource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan monitorV2ResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq, diags := buildMonitorV2CreateRequest(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiResp, err := r.client.CreateMonitorV2(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create monitor, got error: %s", err.Error()))
		return
	}
	if apiResp == nil || apiResp.MonitorID == "" {
		resp.Diagnostics.AddError("API Error", "Monitor creation response did not contain a MonitorID")
		return
	}

	plan.ID = types.StringValue(apiResp.MonitorID)
	if err := r.readMonitorV2IntoState(ctx, apiResp.MonitorID, &plan, &resp.Diagnostics); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read created monitor %s, got error: %s", apiResp.MonitorID, err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *monitorV2Resource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state monitorV2ResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := state.ID.ValueString()
	if err := r.readMonitorV2IntoState(ctx, id, &state, &resp.Diagnostics); err != nil {
		if errors.Is(err, ErrNotFound) {
			tflog.Warn(ctx, fmt.Sprintf("Monitor %s not found, removing from state", id))
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read monitor %s, got error: %s", id, err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *monitorV2Resource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan monitorV2ResourceModel
	var state monitorV2ResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateReq, diags := buildMonitorV2UpdateRequest(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := state.ID.ValueString()
	if err := r.client.UpdateMonitorV2(ctx, id, updateReq); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update monitor %s, got error: %s", id, err.Error()))
		return
	}

	plan.ID = state.ID
	if err := r.readMonitorV2IntoState(ctx, id, &plan, &resp.Diagnostics); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read updated monitor %s, got error: %s", id, err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *monitorV2Resource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state monitorV2ResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := state.ID.ValueString()
	if err := r.client.DeleteMonitorV2(ctx, id); err != nil && !errors.Is(err, ErrNotFound) {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete monitor %s, got error: %s", id, err))
		return
	}
}

func (r *monitorV2Resource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *monitorV2Resource) readMonitorV2IntoState(ctx context.Context, id string, state *monitorV2ResourceModel, diags *diag.Diagnostics) error {
	remoteYaml, err := r.client.GetMonitorV2(ctx, id)
	if err != nil {
		return err
	}

	var remote models.UpdateMonitorRequest
	if err := yaml.Unmarshal(remoteYaml, &remote); err != nil {
		return fmt.Errorf("unable to unmarshal monitor response into typed model: %w", err)
	}

	mapMonitorV2SDKToModel(ctx, id, &remote, state, diags)
	if diags.HasError() {
		return errors.New("failed to map monitor response into Terraform state")
	}
	return nil
}

func buildMonitorV2CreateRequest(ctx context.Context, plan *monitorV2ResourceModel) (*models.CreateMonitorRequest, diag.Diagnostics) {
	var diags diag.Diagnostics
	isProvisioned := true
	req := &models.CreateMonitorRequest{
		Annotations:          monitorV2AnnotationsToSDK(ctx, plan.Annotations, &diags),
		AutoResolve:          monitorV2Bool(plan.AutoResolve),
		Category:             monitorV2String(plan.Category),
		ExecutionErrorState:  monitorV2String(plan.ExecutionErrorState),
		IsPaused:             monitorV2Bool(plan.IsPaused),
		IsProvisioned:        &isProvisioned,
		Labels:               monitorV2StringMap(ctx, plan.Labels, &diags),
		MeasurementType:      monitorV2String(plan.MeasurementType),
		NoDataState:          monitorV2String(plan.NoDataState),
		Routing:              monitorV2StringList(ctx, plan.Routing, &diags),
		Severity:             monitorV2String(plan.Severity),
		Team:                 monitorV2String(plan.Team),
		Title:                monitorV2StringPtr(plan.Title),
		Catalog:              nil,
		Display:              monitorV2DisplayToSDK(ctx, plan.Display, &diags),
		EvaluationInterval:   monitorV2EvaluationIntervalToSDK(plan.EvaluationInterval, &diags),
		Model:                monitorV2ModelToSDK(ctx, plan, &diags),
		NotificationSettings: monitorV2NotificationSettingsToSDK(ctx, plan.NotificationSettings, &diags),
	}
	return req, diags
}

func buildMonitorV2UpdateRequest(ctx context.Context, plan *monitorV2ResourceModel) (*models.UpdateMonitorRequest, diag.Diagnostics) {
	var diags diag.Diagnostics
	req := &models.UpdateMonitorRequest{
		Annotations:          monitorV2AnnotationsToSDK(ctx, plan.Annotations, &diags),
		AutoResolve:          monitorV2Bool(plan.AutoResolve),
		Category:             monitorV2String(plan.Category),
		ExecutionErrorState:  monitorV2String(plan.ExecutionErrorState),
		IsPaused:             monitorV2Bool(plan.IsPaused),
		Labels:               monitorV2StringMap(ctx, plan.Labels, &diags),
		MeasurementType:      monitorV2String(plan.MeasurementType),
		NoDataState:          monitorV2String(plan.NoDataState),
		Routing:              monitorV2StringList(ctx, plan.Routing, &diags),
		Severity:             monitorV2String(plan.Severity),
		Team:                 monitorV2String(plan.Team),
		Title:                monitorV2StringPtr(plan.Title),
		Catalog:              nil,
		Display:              monitorV2DisplayToSDK(ctx, plan.Display, &diags),
		EvaluationInterval:   monitorV2EvaluationIntervalToSDK(plan.EvaluationInterval, &diags),
		Model:                monitorV2ModelToSDK(ctx, plan, &diags),
		NotificationSettings: monitorV2NotificationSettingsToSDK(ctx, plan.NotificationSettings, &diags),
	}
	return req, diags
}

func monitorV2ModelToSDK(ctx context.Context, plan *monitorV2ResourceModel, diags *diag.Diagnostics) *models.Model {
	query := monitorV2QueryToSDK(plan.Query, diags)
	reducers := make([]*models.ReducerModel, 0, len(plan.Reducers))
	for _, reducer := range plan.Reducers {
		reducers = append(reducers, monitorV2ReducerToSDK(reducer, diags))
	}

	thresholds := make([]*models.Threshold, 0, len(plan.Thresholds))
	for i, threshold := range plan.Thresholds {
		thresholds = append(thresholds, monitorV2ThresholdToSDK(ctx, threshold, i, diags))
	}

	return &models.Model{
		Queries:    []*models.BaseQuery{query},
		Reducers:   reducers,
		Thresholds: thresholds,
	}
}

func monitorV2QueryToSDK(query *monitorV2QueryModel, diags *diag.Diagnostics) *models.BaseQuery {
	if query == nil {
		return nil
	}

	name := monitorV2String(query.Name)
	if name == "" {
		name = monitorV2DefaultQueryName
	}

	req := &models.BaseQuery{
		Name:              name,
		Expression:        monitorV2String(query.Expression),
		DatasourceID:      monitorV2String(query.DatasourceID),
		RelativeTimerange: monitorV2RelativeRangeToSDK(query.RelativeTimerange, diags),
	}

	switch monitorV2String(query.Type) {
	case monitorV2QueryTypeGCQL:
		req.DataType = monitorV2String(query.DataType)
		if parsed, ok := monitorV2ParseDuration(query.InstantRollup, path.Root("query").AtName("instant_rollup"), diags); ok {
			req.InstantRollup = monitorV2DurationToString(parsed)
		}
	case monitorV2QueryTypeMetricsQL:
		req.DatasourceType = monitorV2String(query.DatasourceType)
		if req.DatasourceType == "" {
			req.DatasourceType = monitorV2DatasourcePrometheus
		}
		req.QueryType = monitorV2String(query.QueryType)
		if req.QueryType == "" {
			req.QueryType = monitorV2QueryTypeInstant
		}
		req.Rollup = monitorV2RollupToSDK(query.Rollup, diags)
	case monitorV2QueryTypeRawSQL:
		req.DatasourceType = monitorV2DatasourceClickhouse
		req.QueryType = monitorV2String(query.QueryType)
		if req.QueryType == "" {
			req.QueryType = monitorV2QueryTypeInstant
		}
	}

	return req
}

func monitorV2RollupToSDK(rollup *monitorV2RollupModel, diags *diag.Diagnostics) *models.Rollup {
	if rollup == nil {
		return nil
	}
	parsed, ok := monitorV2ParseDuration(rollup.Time, path.Root("query").AtName("rollup").AtName("time"), diags)
	if !ok {
		return nil
	}
	duration := models.Duration(parsed)
	return &models.Rollup{
		Function: monitorV2String(rollup.Function),
		Time:     duration,
	}
}

func monitorV2ReducerToSDK(reducer monitorV2ReducerModel, diags *diag.Diagnostics) *models.ReducerModel {
	return &models.ReducerModel{
		Expression:        monitorV2String(reducer.Expression),
		InputName:         monitorV2String(reducer.InputName),
		Name:              monitorV2String(reducer.Name),
		Type:              monitorV2StringPtr(reducer.Type),
		RelativeTimerange: monitorV2RelativeRangeToSDK(reducer.RelativeTimerange, diags),
	}
}

func monitorV2ThresholdToSDK(ctx context.Context, threshold monitorV2ThresholdModel, index int, diags *diag.Diagnostics) *models.Threshold {
	parentOp := monitorV2String(threshold.Operator)
	if threshold.CustomResolveThreshold != nil && parentOp != "" && !monitorV2SupportsCustomResolveOperator(parentOp) {
		diags.AddAttributeError(
			path.Root("threshold").AtListIndex(index).AtName("operator"),
			"Unsupported threshold operator for custom resolve threshold",
			"`custom_resolve_threshold` is supported only when threshold.operator is one of `gt`, `lt`, `within_range`, or `outside_range`.",
		)
	}

	return &models.Threshold{
		InputName:              monitorV2StringPtr(threshold.InputName),
		Name:                   monitorV2StringPtr(threshold.Name),
		Operator:               monitorV2StringPtr(threshold.Operator),
		Values:                 monitorV2Float64List(ctx, threshold.Values, diags),
		RelativeTimerange:      monitorV2RelativeRangeToSDK(threshold.RelativeTimerange, diags),
		CustomResolveThreshold: monitorV2CustomResolveThresholdToSDK(ctx, threshold.CustomResolveThreshold, diags),
	}
}

func monitorV2CustomResolveThresholdToSDK(ctx context.Context, threshold *monitorV2CustomResolveThresholdModel, diags *diag.Diagnostics) *models.CustomResolveThreshold {
	if threshold == nil {
		return nil
	}
	return &models.CustomResolveThreshold{
		Operator: monitorV2StringPtr(threshold.Operator),
		Values:   monitorV2Float64List(ctx, threshold.Values, diags),
	}
}

func monitorV2EvaluationIntervalToSDK(interval *monitorV2EvaluationIntervalModel, diags *diag.Diagnostics) *models.EvaluationInterval {
	if interval == nil {
		return nil
	}

	req := &models.EvaluationInterval{}
	if parsed, ok := monitorV2ParseDuration(interval.Interval, path.Root("evaluation_interval").AtName("interval"), diags); ok {
		req.Interval = strfmt.Duration(parsed)
	}
	if parsed, ok := monitorV2ParseDuration(interval.PendingFor, path.Root("evaluation_interval").AtName("pending_for"), diags); ok {
		pendingFor := models.Duration(parsed)
		req.PendingFor = &pendingFor
	}
	return req
}

func monitorV2DisplayToSDK(ctx context.Context, display *monitorV2DisplayModel, diags *diag.Diagnostics) *models.DisplayModel {
	if display == nil {
		return nil
	}
	return &models.DisplayModel{
		ContextHeaderLabels:  monitorV2StringList(ctx, display.ContextHeaderLabels, diags),
		Description:          monitorV2String(display.Description),
		Header:               monitorV2String(display.Header),
		ResourceHeaderLabels: monitorV2StringList(ctx, display.ResourceHeaderLabels, diags),
		TemplateLanguage:     monitorV2String(display.TemplateLanguage),
	}
}

func monitorV2NotificationSettingsToSDK(ctx context.Context, settings *monitorV2NotificationSettingsModel, diags *diag.Diagnostics) *models.NotificationSettings {
	if settings == nil {
		return nil
	}
	req := &models.NotificationSettings{
		ConnectedApps:         monitorV2StringList(ctx, settings.ConnectedApps, diags),
		ConnectedAppParams:    monitorV2ConnectedAppParamsToSDK(ctx, settings.ConnectedAppParams, diags),
		DisableRenotification: monitorV2Bool(settings.DisableRenotification),
		Method:                monitorV2String(settings.Method),
		StatusFilters:         monitorV2IssueStatuses(ctx, settings.StatusFilters, diags),
	}
	if interval := monitorV2String(settings.RenotificationInterval); interval != "" {
		req.RenotificationInterval = models.RenotificationDuration(interval)
	}
	return req
}

func monitorV2RelativeRangeToSDK(relativeRange *monitorV2RelativeRangeModel, diags *diag.Diagnostics) *models.RelativeTimerange {
	if relativeRange == nil {
		return nil
	}
	req := &models.RelativeTimerange{}
	if parsed, ok := monitorV2ParseDuration(relativeRange.From, path.Root("relative_timerange").AtName("from"), diags); ok {
		req.From = strfmt.Duration(parsed)
	}
	if parsed, ok := monitorV2ParseDuration(relativeRange.To, path.Root("relative_timerange").AtName("to"), diags); ok {
		req.To = strfmt.Duration(parsed)
	}
	return req
}

func mapMonitorV2SDKToModel(ctx context.Context, id string, remote *models.UpdateMonitorRequest, state *monitorV2ResourceModel, diags *diag.Diagnostics) {
	previousQuery := state.Query
	previousEvaluationInterval := state.EvaluationInterval
	previousNotificationSettings := state.NotificationSettings
	previousReducers := state.Reducers
	previousThresholds := state.Thresholds

	state.ID = types.StringValue(id)
	state.Title = monitorV2StringPtrToType(remote.Title)
	state.Severity = monitorV2NullableString(remote.Severity)
	state.MeasurementType = monitorV2NullableString(remote.MeasurementType)
	state.ExecutionErrorState = monitorV2NullableString(remote.ExecutionErrorState)
	state.NoDataState = monitorV2NullableString(remote.NoDataState)
	state.IsPaused = types.BoolValue(remote.IsPaused)
	state.AutoResolve = types.BoolValue(remote.AutoResolve)
	state.Category = monitorV2NullableString(remote.Category)
	state.Team = monitorV2NullableString(remote.Team)
	state.Labels = monitorV2MapType(ctx, remote.Labels, diags)
	state.Annotations = monitorV2MapType(ctx, monitorV2FilterAnnotations(remote.Annotations), diags)
	state.Routing = monitorV2StringListType(ctx, remote.Routing, diags)
	state.Display = monitorV2DisplayFromSDK(ctx, remote.Display, diags)
	state.EvaluationInterval = monitorV2PreserveEvaluationIntervalDurations(previousEvaluationInterval, monitorV2EvaluationIntervalFromSDK(remote.EvaluationInterval))
	state.NotificationSettings = monitorV2PreserveNotificationSettingsDurations(previousNotificationSettings, monitorV2NotificationSettingsFromSDK(ctx, remote.NotificationSettings, diags))

	if remote.Model == nil {
		state.Query = nil
		state.Reducers = nil
		state.Thresholds = nil
		return
	}
	if len(remote.Model.Queries) > 0 && remote.Model.Queries[0] != nil {
		state.Query = monitorV2PreserveQueryDurations(previousQuery, monitorV2QueryFromSDK(remote.Model.Queries[0], remote.Annotations))
	}
	state.Reducers = monitorV2PreserveReducerDurations(previousReducers, monitorV2ReducersFromSDK(remote.Model.Reducers))
	state.Thresholds = monitorV2PreserveThresholdDurations(previousThresholds, monitorV2ThresholdsFromSDK(ctx, remote.Model.Thresholds, diags))
}

func monitorV2PreserveQueryDurations(previous, updated *monitorV2QueryModel) *monitorV2QueryModel {
	if previous == nil || updated == nil {
		return updated
	}

	updated.InstantRollup = monitorV2PreserveDurationString(previous.InstantRollup, updated.InstantRollup)
	updated.RelativeTimerange = monitorV2PreserveRelativeRangeDurations(previous.RelativeTimerange, updated.RelativeTimerange)
	if previous.Rollup != nil && updated.Rollup != nil {
		updated.Rollup.Time = monitorV2PreserveDurationString(previous.Rollup.Time, updated.Rollup.Time)
	}
	return updated
}

func monitorV2PreserveEvaluationIntervalDurations(previous, updated *monitorV2EvaluationIntervalModel) *monitorV2EvaluationIntervalModel {
	if previous == nil || updated == nil {
		return updated
	}

	updated.Interval = monitorV2PreserveDurationString(previous.Interval, updated.Interval)
	updated.PendingFor = monitorV2PreserveDurationString(previous.PendingFor, updated.PendingFor)
	return updated
}

func monitorV2PreserveNotificationSettingsDurations(previous, updated *monitorV2NotificationSettingsModel) *monitorV2NotificationSettingsModel {
	if previous == nil || updated == nil {
		return updated
	}

	updated.RenotificationInterval = monitorV2PreserveDurationString(previous.RenotificationInterval, updated.RenotificationInterval)
	return updated
}

func monitorV2PreserveReducerDurations(previous, updated []monitorV2ReducerModel) []monitorV2ReducerModel {
	for i := range updated {
		if i >= len(previous) {
			break
		}
		updated[i].RelativeTimerange = monitorV2PreserveRelativeRangeDurations(previous[i].RelativeTimerange, updated[i].RelativeTimerange)
	}
	return updated
}

func monitorV2PreserveThresholdDurations(previous, updated []monitorV2ThresholdModel) []monitorV2ThresholdModel {
	for i := range updated {
		if i >= len(previous) {
			break
		}
		updated[i].RelativeTimerange = monitorV2PreserveRelativeRangeDurations(previous[i].RelativeTimerange, updated[i].RelativeTimerange)
	}
	return updated
}

func monitorV2PreserveRelativeRangeDurations(previous, updated *monitorV2RelativeRangeModel) *monitorV2RelativeRangeModel {
	if previous == nil || updated == nil {
		return updated
	}

	updated.From = monitorV2PreserveDurationString(previous.From, updated.From)
	updated.To = monitorV2PreserveDurationString(previous.To, updated.To)
	return updated
}

func monitorV2PreserveDurationString(previous, updated types.String) types.String {
	if previous.IsNull() || previous.IsUnknown() || updated.IsNull() || updated.IsUnknown() {
		return updated
	}
	if monitorV2DurationStringsEqual(previous.ValueString(), updated.ValueString()) {
		return previous
	}
	return updated
}

func monitorV2DurationStringsEqual(left, right string) bool {
	leftNormalized, leftOK := monitorV2NormalizeDurationString(left)
	rightNormalized, rightOK := monitorV2NormalizeDurationString(right)
	return leftOK && rightOK && leftNormalized == rightNormalized
}

func monitorV2QueryFromSDK(query *models.BaseQuery, annotations map[string]string) *monitorV2QueryModel {
	return &monitorV2QueryModel{
		Name:              monitorV2NullableString(query.Name),
		Type:              types.StringValue(monitorV2QueryTypeFromSDK(query, annotations)),
		Expression:        monitorV2NullableString(query.Expression),
		DataType:          monitorV2NullableString(query.DataType),
		DatasourceType:    monitorV2NullableString(query.DatasourceType),
		DatasourceID:      monitorV2NullableString(query.DatasourceID),
		QueryType:         monitorV2NullableString(query.QueryType),
		InstantRollup:     monitorV2DurationStringToType(query.InstantRollup),
		Rollup:            monitorV2RollupFromSDK(query.Rollup),
		RelativeTimerange: monitorV2RelativeRangeFromSDK(query.RelativeTimerange),
	}
}

func monitorV2QueryTypeFromSDK(query *models.BaseQuery, annotations map[string]string) string {
	if queryType := annotations[monitorV2QueryTypeAnnotationKey]; monitorV2IsValidQueryType(queryType) {
		return queryType
	}
	if query.DataType != "" {
		return monitorV2QueryTypeGCQL
	}
	if query.Rollup != nil || query.DatasourceType == monitorV2DatasourcePrometheus || query.DatasourceType == monitorV2DatasourceMetrics {
		return monitorV2QueryTypeMetricsQL
	}
	return monitorV2QueryTypeRawSQL
}

func monitorV2IsValidQueryType(queryType string) bool {
	switch queryType {
	case monitorV2QueryTypeGCQL, monitorV2QueryTypeMetricsQL, monitorV2QueryTypeRawSQL:
		return true
	default:
		return false
	}
}

func monitorV2RollupFromSDK(rollup *models.Rollup) *monitorV2RollupModel {
	if rollup == nil {
		return nil
	}
	return &monitorV2RollupModel{
		Function: monitorV2NullableString(rollup.Function),
		Time:     monitorV2DurationToType(time.Duration(rollup.Time)),
	}
}

func monitorV2ReducersFromSDK(reducers []*models.ReducerModel) []monitorV2ReducerModel {
	if len(reducers) == 0 {
		return nil
	}
	result := make([]monitorV2ReducerModel, 0, len(reducers))
	for _, reducer := range reducers {
		if reducer == nil {
			continue
		}
		result = append(result, monitorV2ReducerModel{
			Name:              monitorV2NullableString(reducer.Name),
			InputName:         monitorV2NullableString(reducer.InputName),
			Type:              monitorV2StringPtrToType(reducer.Type),
			Expression:        monitorV2NullableString(reducer.Expression),
			RelativeTimerange: monitorV2RelativeRangeFromSDK(reducer.RelativeTimerange),
		})
	}
	return result
}

func monitorV2ThresholdsFromSDK(ctx context.Context, thresholds []*models.Threshold, diags *diag.Diagnostics) []monitorV2ThresholdModel {
	if len(thresholds) == 0 {
		return nil
	}
	result := make([]monitorV2ThresholdModel, 0, len(thresholds))
	for _, threshold := range thresholds {
		if threshold == nil {
			continue
		}
		result = append(result, monitorV2ThresholdModel{
			Name:                   monitorV2StringPtrToType(threshold.Name),
			InputName:              monitorV2StringPtrToType(threshold.InputName),
			Operator:               monitorV2StringPtrToType(threshold.Operator),
			Values:                 monitorV2Float64ListType(ctx, threshold.Values, diags),
			RelativeTimerange:      monitorV2RelativeRangeFromSDK(threshold.RelativeTimerange),
			CustomResolveThreshold: monitorV2CustomResolveThresholdFromSDK(ctx, threshold.CustomResolveThreshold, diags),
		})
	}
	return result
}

func monitorV2CustomResolveThresholdFromSDK(ctx context.Context, threshold *models.CustomResolveThreshold, diags *diag.Diagnostics) *monitorV2CustomResolveThresholdModel {
	if threshold == nil {
		return nil
	}
	return &monitorV2CustomResolveThresholdModel{
		Operator: monitorV2StringPtrToType(threshold.Operator),
		Values:   monitorV2Float64ListType(ctx, threshold.Values, diags),
	}
}

func monitorV2EvaluationIntervalFromSDK(interval *models.EvaluationInterval) *monitorV2EvaluationIntervalModel {
	if interval == nil {
		return nil
	}
	pendingFor := types.StringNull()
	if interval.PendingFor != nil {
		pendingFor = monitorV2DurationToType(time.Duration(*interval.PendingFor))
	}
	return &monitorV2EvaluationIntervalModel{
		Interval:   monitorV2DurationToType(time.Duration(interval.Interval)),
		PendingFor: pendingFor,
	}
}

func monitorV2DisplayFromSDK(ctx context.Context, display *models.DisplayModel, diags *diag.Diagnostics) *monitorV2DisplayModel {
	if display == nil {
		return nil
	}
	return &monitorV2DisplayModel{
		Header:               monitorV2NullableString(display.Header),
		Description:          monitorV2NullableString(display.Description),
		ResourceHeaderLabels: monitorV2StringListType(ctx, display.ResourceHeaderLabels, diags),
		ContextHeaderLabels:  monitorV2StringListType(ctx, display.ContextHeaderLabels, diags),
		TemplateLanguage:     monitorV2NullableString(display.TemplateLanguage),
	}
}

func monitorV2NotificationSettingsFromSDK(ctx context.Context, settings *models.NotificationSettings, diags *diag.Diagnostics) *monitorV2NotificationSettingsModel {
	if settings == nil {
		return nil
	}
	return &monitorV2NotificationSettingsModel{
		Method:                 monitorV2NullableString(settings.Method),
		ConnectedApps:          monitorV2StringListType(ctx, settings.ConnectedApps, diags),
		ConnectedAppParams:     monitorV2ConnectedAppParamsType(ctx, settings.ConnectedAppParams, diags),
		StatusFilters:          monitorV2IssueStatusListType(ctx, settings.StatusFilters, diags),
		DisableRenotification:  types.BoolValue(settings.DisableRenotification),
		RenotificationInterval: monitorV2AnyString(settings.RenotificationInterval),
	}
}

func monitorV2RelativeRangeFromSDK(relativeRange *models.RelativeTimerange) *monitorV2RelativeRangeModel {
	if relativeRange == nil {
		return nil
	}
	return &monitorV2RelativeRangeModel{
		From: monitorV2DurationToType(time.Duration(relativeRange.From)),
		To:   monitorV2DurationToType(time.Duration(relativeRange.To)),
	}
}

func monitorV2String(value types.String) string {
	if value.IsNull() || value.IsUnknown() {
		return ""
	}
	return value.ValueString()
}

func monitorV2Bool(value types.Bool) bool {
	if value.IsNull() || value.IsUnknown() {
		return false
	}
	return value.ValueBool()
}

func monitorV2StringPtr(value types.String) *string {
	str := monitorV2String(value)
	if str == "" {
		return nil
	}
	return &str
}

func monitorV2StringPtrToType(value *string) types.String {
	if value == nil || *value == "" {
		return types.StringNull()
	}
	return types.StringValue(*value)
}

func monitorV2SupportsCustomResolveOperator(operator string) bool {
	switch operator {
	case "gt", "lt", "within_range", "outside_range":
		return true
	default:
		return false
	}
}

func monitorV2NullableString(value string) types.String {
	if value == "" {
		return types.StringNull()
	}
	return types.StringValue(value)
}

func monitorV2AnyString(value any) types.String {
	if value == nil {
		return types.StringNull()
	}
	str := fmt.Sprint(value)
	if str == "" || str == "<nil>" {
		return types.StringNull()
	}
	return types.StringValue(normalizeTimeString(str))
}

func monitorV2StringMap(ctx context.Context, value types.Map, diags *diag.Diagnostics) map[string]string {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}
	result := make(map[string]string)
	diags.Append(value.ElementsAs(ctx, &result, false)...)
	return result
}

func monitorV2AnnotationsToSDK(ctx context.Context, annotations types.Map, diags *diag.Diagnostics) map[string]string {
	return monitorV2FilterAnnotations(monitorV2StringMap(ctx, annotations, diags))
}

func monitorV2StringList(ctx context.Context, value types.List, diags *diag.Diagnostics) []string {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}
	var result []string
	diags.Append(value.ElementsAs(ctx, &result, false)...)
	return result
}

func monitorV2Float64List(ctx context.Context, value types.List, diags *diag.Diagnostics) []float64 {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}
	var result []float64
	diags.Append(value.ElementsAs(ctx, &result, false)...)
	return result
}

func monitorV2IssueStatuses(ctx context.Context, value types.List, diags *diag.Diagnostics) []models.IssueStatus {
	statuses := monitorV2StringList(ctx, value, diags)
	if len(statuses) == 0 {
		return nil
	}
	result := make([]models.IssueStatus, 0, len(statuses))
	for _, status := range statuses {
		result = append(result, models.IssueStatus(status))
	}
	return result
}

func monitorV2ConnectedAppParamsToSDK(ctx context.Context, value types.Map, diags *diag.Diagnostics) models.ConnectedAppParams {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}

	var params map[string]monitorV2ConnectedAppDeliveryOptionsModel
	diags.Append(value.ElementsAs(ctx, &params, false)...)
	if len(params) == 0 {
		return nil
	}

	result := make(models.ConnectedAppParams, len(params))
	for appID, options := range params {
		result[appID] = models.ConnectedAppDeliveryOptions{
			Channels: monitorV2StringList(ctx, options.Channels, diags),
		}
	}
	return result
}

func monitorV2MapType(ctx context.Context, value map[string]string, diags *diag.Diagnostics) types.Map {
	if len(value) == 0 {
		return types.MapNull(types.StringType)
	}
	result, mapDiags := types.MapValueFrom(ctx, types.StringType, value)
	diags.Append(mapDiags...)
	return result
}

func monitorV2StringListType(ctx context.Context, value []string, diags *diag.Diagnostics) types.List {
	if len(value) == 0 {
		return types.ListNull(types.StringType)
	}
	result, listDiags := types.ListValueFrom(ctx, types.StringType, value)
	diags.Append(listDiags...)
	return result
}

func monitorV2Float64ListType(ctx context.Context, value []float64, diags *diag.Diagnostics) types.List {
	if len(value) == 0 {
		return types.ListNull(types.Float64Type)
	}
	result, listDiags := types.ListValueFrom(ctx, types.Float64Type, value)
	diags.Append(listDiags...)
	return result
}

func monitorV2IssueStatusListType(ctx context.Context, value []models.IssueStatus, diags *diag.Diagnostics) types.List {
	if len(value) == 0 {
		return types.ListNull(types.StringType)
	}
	statuses := make([]string, 0, len(value))
	for _, status := range value {
		statuses = append(statuses, string(status))
	}
	return monitorV2StringListType(ctx, statuses, diags)
}

func monitorV2ConnectedAppParamsType(ctx context.Context, value models.ConnectedAppParams, diags *diag.Diagnostics) types.Map {
	attrTypes := monitorV2ConnectedAppDeliveryOptionsAttrTypes()
	objectType := types.ObjectType{AttrTypes: attrTypes}
	if len(value) == 0 {
		return types.MapNull(objectType)
	}

	elements := make(map[string]attr.Value, len(value))
	for appID, options := range value {
		channels := monitorV2StringListType(ctx, options.Channels, diags)
		object, objectDiags := types.ObjectValue(attrTypes, map[string]attr.Value{
			"channels": channels,
		})
		diags.Append(objectDiags...)
		elements[appID] = object
	}

	result, mapDiags := types.MapValue(objectType, elements)
	diags.Append(mapDiags...)
	return result
}

func monitorV2ConnectedAppDeliveryOptionsAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"channels": types.ListType{ElemType: types.StringType},
	}
}

func monitorV2ParseDuration(value types.String, attrPath path.Path, diags *diag.Diagnostics) (time.Duration, bool) {
	raw := monitorV2String(value)
	if raw == "" {
		return 0, false
	}

	normalized := monitorV2NormalizeDurationForParse(raw)
	parsed, err := strfmt.ParseDuration(normalized)
	if err != nil {
		diags.AddAttributeError(
			attrPath,
			"Invalid duration",
			fmt.Sprintf("Expected a valid duration such as `5m`, `1h`, or `1 day`; got `%s`: %s", raw, err),
		)
		return 0, false
	}
	return parsed, true
}

func monitorV2NormalizeDurationForParse(value string) string {
	return strings.TrimSpace(normalizeDayDurations(normalizeHumanDurations(value)))
}

func monitorV2NormalizeDurationString(value string) (string, bool) {
	raw := strings.TrimSpace(value)
	if raw == "" {
		return "", false
	}

	parsed, err := strfmt.ParseDuration(monitorV2NormalizeDurationForParse(raw))
	if err != nil {
		return "", false
	}
	return monitorV2DurationToString(parsed), true
}

func monitorV2DurationStringToType(value string) types.String {
	normalized, ok := monitorV2NormalizeDurationString(value)
	if !ok {
		return monitorV2NullableString(value)
	}
	return types.StringValue(normalized)
}

func monitorV2DurationToString(value time.Duration) string {
	if value == 0 {
		return "0m"
	}
	return normalizeTimeString(value.String())
}

func monitorV2DurationToType(value time.Duration) types.String {
	return types.StringValue(monitorV2DurationToString(value))
}

func monitorV2FilterAnnotations(annotations map[string]string) map[string]string {
	if len(annotations) == 0 {
		return nil
	}
	filtered := make(map[string]string, len(annotations))
	for key, value := range annotations {
		if monitorV2IsInternalAnnotation(key) {
			continue
		}
		filtered[key] = value
	}
	if len(filtered) == 0 {
		return nil
	}
	return filtered
}

func monitorV2IsInternalAnnotation(key string) bool {
	switch key {
	case monitorV2DataTypeAnnotationKey, monitorV2QueryTypeAnnotationKey:
		return true
	default:
		return false
	}
}
