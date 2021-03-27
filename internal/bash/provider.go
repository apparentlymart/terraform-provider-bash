package bash

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
	"github.com/hashicorp/terraform-plugin-go/tfprotov5/tftypes"
)

type Provider struct {
}

func NewProvider() tfprotov5.ProviderServer {
	return &Provider{}
}

func (p *Provider) GetProviderSchema(ctx context.Context, req *tfprotov5.GetProviderSchemaRequest) (*tfprotov5.GetProviderSchemaResponse, error) {
	return &tfprotov5.GetProviderSchemaResponse{
		Provider: &tfprotov5.Schema{
			Block: &tfprotov5.SchemaBlock{},
		},
		DataSourceSchemas: map[string]*tfprotov5.Schema{
			"bash_script": {
				Block: &tfprotov5.SchemaBlock{
					Attributes: []*tfprotov5.SchemaAttribute{
						{
							Name:            "source",
							Type:            tftypes.String,
							Required:        true,
							Description:     "Bash source code for the body of the script, which may use any of the variables declared in the `variables` argument via the usual bash variable syntax.",
							DescriptionKind: tfprotov5.StringKindMarkdown,
						},
						{
							Name:            "variables",
							Type:            tftypes.DynamicPseudoType,
							Optional:        true,
							Description:     "An object describing the variables to present to the script, where each attribute translates to one bash variable.",
							DescriptionKind: tfprotov5.StringKindMarkdown,
						},
						{
							Name:            "result",
							Type:            tftypes.String,
							Computed:        true,
							Description:     "The resulting script, which combines the script body given in `source` with the variables given in `variables`.",
							DescriptionKind: tfprotov5.StringKindMarkdown,
						},
					},
				},
			},
		},
	}, nil
}

func (p *Provider) PrepareProviderConfig(ctx context.Context, req *tfprotov5.PrepareProviderConfigRequest) (*tfprotov5.PrepareProviderConfigResponse, error) {
	// This provider has an empty provider configuration schema, so we have
	// nothing to do here except echo back the empty object we were given.
	return &tfprotov5.PrepareProviderConfigResponse{
		PreparedConfig: req.Config,
	}, nil
}

func (p *Provider) ConfigureProvider(ctx context.Context, req *tfprotov5.ConfigureProviderRequest) (*tfprotov5.ConfigureProviderResponse, error) {
	// This provider has an empty provider configuration schema, so there's
	// nothing to do here.
	return &tfprotov5.ConfigureProviderResponse{}, nil
}

func (p *Provider) StopProvider(ctx context.Context, req *tfprotov5.StopProviderRequest) (*tfprotov5.StopProviderResponse, error) {
	// We have no long-running operations, so nothing to stop.
	return &tfprotov5.StopProviderResponse{}, nil
}

func (p *Provider) ValidateDataSourceConfig(ctx context.Context, req *tfprotov5.ValidateDataSourceConfigRequest) (*tfprotov5.ValidateDataSourceConfigResponse, error) {
	if req.TypeName != "bash_script" {
		// Should never get here because we have no other data resource types
		// declared in the schema.
		return nil, fmt.Errorf("unsupported data resource type %s", req.TypeName)
	}

	_, diags := newBashScriptConfig(req.Config)

	return &tfprotov5.ValidateDataSourceConfigResponse{
		Diagnostics: diags,
	}, nil
}

func (p *Provider) ReadDataSource(ctx context.Context, req *tfprotov5.ReadDataSourceRequest) (*tfprotov5.ReadDataSourceResponse, error) {
	if req.TypeName != "bash_script" {
		// Should never get here because we have no other data resource types
		// declared in the schema.
		return nil, fmt.Errorf("unsupported data resource type %s", req.TypeName)
	}

	var diags []*tfprotov5.Diagnostic

	config, diags := newBashScriptConfig(req.Config)
	if len(diags) != 0 {
		// NOTE: This assumes that diags doesn't contain any warnings, which
		// is always true at the time of writing this.
		return &tfprotov5.ReadDataSourceResponse{
			Diagnostics: diags,
		}, nil
	}

	varDecls := variablesToBashDecls(config.Variables)
	// TODO: varDecls should actually get merged with the user's given source
	// code.
	ret := config.ResultDynamicValue(varDecls)

	return &tfprotov5.ReadDataSourceResponse{
		State:       ret,
		Diagnostics: diags,
	}, nil
}

func (p *Provider) ValidateResourceTypeConfig(ctx context.Context, req *tfprotov5.ValidateResourceTypeConfigRequest) (*tfprotov5.ValidateResourceTypeConfigResponse, error) {
	return nil, fmt.Errorf("unsupported managed resource type %s", req.TypeName)
}

func (p *Provider) UpgradeResourceState(ctx context.Context, req *tfprotov5.UpgradeResourceStateRequest) (*tfprotov5.UpgradeResourceStateResponse, error) {
	return nil, fmt.Errorf("unsupported managed resource type %s", req.TypeName)
}

func (p *Provider) ReadResource(ctx context.Context, req *tfprotov5.ReadResourceRequest) (*tfprotov5.ReadResourceResponse, error) {
	return nil, fmt.Errorf("unsupported managed resource type %s", req.TypeName)
}

func (p *Provider) PlanResourceChange(ctx context.Context, req *tfprotov5.PlanResourceChangeRequest) (*tfprotov5.PlanResourceChangeResponse, error) {
	return nil, fmt.Errorf("unsupported managed resource type %s", req.TypeName)
}

func (p *Provider) ApplyResourceChange(ctx context.Context, req *tfprotov5.ApplyResourceChangeRequest) (*tfprotov5.ApplyResourceChangeResponse, error) {
	return nil, fmt.Errorf("unsupported managed resource type %s", req.TypeName)
}

func (p *Provider) ImportResourceState(ctx context.Context, req *tfprotov5.ImportResourceStateRequest) (*tfprotov5.ImportResourceStateResponse, error) {
	return nil, fmt.Errorf("unsupported managed resource type %s", req.TypeName)
}
