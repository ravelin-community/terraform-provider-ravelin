package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ provider.Provider = &ravelinProvider{}

type ravelinProvider struct {
	version string
	project string
}

type ravelinProviderModel struct {
	Project types.String `tfsdk:"project"`
}

func (p *ravelinProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "ravelin"
	resp.Version = p.version
}

func (p *ravelinProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"project": schema.StringAttribute{
				MarkdownDescription: "GCP project name used by default for all resources",
				Optional:            true,
			},
		},
	}
}

func (p *ravelinProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config ravelinProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if !config.Project.IsNull() {
		p.project = config.Project.ValueString()
	}

	// Make the provider available to data sources and resources
	resp.DataSourceData = p
	resp.ResourceData = p
}

func (p *ravelinProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		func() resource.Resource {
			return &ImageSyncResource{}
		},
	}
}

func (p *ravelinProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		func() datasource.DataSource {
			return &ServiceAgentsDataSource{}
		},
		func() datasource.DataSource {
			return &ConditionalBindingsDataSource{}
		},
		func() datasource.DataSource {
			return &GsudoEscalationsDataSource{}
		},
	}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &ravelinProvider{
			version: version,
		}
	}
}
