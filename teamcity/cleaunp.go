package teamcity

import (
	"context"
	"github.com/hashicorp/terraform-plugin-framework-validators/schemavalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"terraform-provider-teamcity/client"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource              = &cleanupResource{}
	_ resource.ResourceWithConfigure = &cleanupResource{}
)

func NewCleanupResource() resource.Resource {
	return &cleanupResource{}
}

type cleanupResource struct {
	client *client.Client
}

type cleanupResourceModel struct {
	ID          types.String        `tfsdk:"id"`
	Enabled     types.Bool          `tfsdk:"enabled"`
	MaxDuration types.Int64         `tfsdk:"max_duration"`
	Daily       *dailyResourceModel `tfsdk:"daily"`
	Cron        *cronResourceModel  `tfsdk:"cron"`
}

type dailyResourceModel struct {
	Hour   types.Int64 `tfsdk:"hour"`
	Minute types.Int64 `tfsdk:"minute"`
}

type cronResourceModel struct {
	Minute  types.String `tfsdk:"minute"`
	Hour    types.String `tfsdk:"hour"`
	Day     types.String `tfsdk:"day"`
	Month   types.String `tfsdk:"month"`
	DayWeek types.String `tfsdk:"day_week"`
}

func (r *cleanupResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cleanup"
}

func (r *cleanupResource) GetSchema(_ context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		Attributes: map[string]tfsdk.Attribute{
			"id": {
				Type:     types.StringType,
				Computed: true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					resource.UseStateForUnknown(),
				},
			},
			"enabled": {
				Type:     types.BoolType,
				Required: true,
			},
			"max_duration": {
				Type:     types.Int64Type,
				Required: true,
			},
			"daily": {
				Optional: true,
				Attributes: tfsdk.SingleNestedAttributes(map[string]tfsdk.Attribute{
					"hour": {
						Type:     types.Int64Type,
						Required: true,
					},
					"minute": {
						Type:     types.Int64Type,
						Required: true,
					},
				}),
			},
			"cron": {
				Optional: true,
				Attributes: tfsdk.SingleNestedAttributes(map[string]tfsdk.Attribute{
					"minute": {
						Type:     types.StringType,
						Required: true,
					},
					"hour": {
						Type:     types.StringType,
						Required: true,
					},
					"day": {
						Type:     types.StringType,
						Required: true,
					},
					"month": {
						Type:     types.StringType,
						Required: true,
					},
					"day_week": {
						Type:     types.StringType,
						Required: true,
					},
				}),

				Validators: []tfsdk.AttributeValidator{
					schemavalidator.ExactlyOneOf(
						path.MatchRoot("daily"),
						path.MatchRoot("cron"),
					),
				},
			},
		},
	}, nil
}

func (r *cleanupResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*client.Client)
}

func (r *cleanupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan cleanupResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	newState, err := r.update(plan)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error setting cleanup",
			"Cannot set cleanup, unexpected error: "+err.Error(),
		)
		return
	}

	diags = resp.State.Set(ctx, newState)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *cleanupResource) Read(ctx context.Context, _ resource.ReadRequest, resp *resource.ReadResponse) {
	result, err := r.client.GetCleanup()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Cleanup",
			"Could not read cleanup settings: "+err.Error(),
		)
		return
	}

	newState := r.readState(result)
	diags := resp.State.Set(ctx, newState)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *cleanupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan cleanupResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	newState, err := r.update(plan)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error setting cleanup",
			"Cannot set cleanup, unexpected error: "+err.Error(),
		)
		return
	}

	diags = resp.State.Set(ctx, newState)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *cleanupResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
}

func (r *cleanupResource) update(plan cleanupResourceModel) (cleanupResourceModel, error) {
	settings := client.CleanupSettings{
		Enabled:     plan.Enabled.ValueBool(),
		MaxDuration: int(plan.MaxDuration.ValueInt64()),
	}

	if plan.Daily != nil {
		settings.Daily = &client.CleanupDaily{
			Hour:   int(plan.Daily.Hour.ValueInt64()),
			Minute: int(plan.Daily.Minute.ValueInt64()),
		}
	}
	if plan.Cron != nil {
		settings.Cron = &client.CleanupCron{
			Minute:  plan.Cron.Minute.ValueString(),
			Hour:    plan.Cron.Hour.ValueString(),
			Day:     plan.Cron.Day.ValueString(),
			Month:   plan.Cron.Month.ValueString(),
			DayWeek: plan.Cron.DayWeek.ValueString(),
		}
	}

	result, err := r.client.SetCleanup(settings)
	if err != nil {
		return cleanupResourceModel{}, err
	}

	return r.readState(result), nil
}

func (r *cleanupResource) readState(result client.CleanupSettings) cleanupResourceModel {
	var state cleanupResourceModel

	state.ID = types.StringValue("placeholder")
	state.Enabled = types.BoolValue(result.Enabled)
	state.MaxDuration = types.Int64Value(int64(result.MaxDuration))

	if result.Daily != nil {
		state.Daily = &dailyResourceModel{
			Hour:   types.Int64Value(int64(result.Daily.Hour)),
			Minute: types.Int64Value(int64(result.Daily.Minute)),
		}
	}
	if result.Cron != nil {
		state.Cron = &cronResourceModel{
			Minute:  types.StringValue(result.Cron.Minute),
			Hour:    types.StringValue(result.Cron.Hour),
			Day:     types.StringValue(result.Cron.Day),
			Month:   types.StringValue(result.Cron.Month),
			DayWeek: types.StringValue(result.Cron.DayWeek),
		}
	}

	return state
}
