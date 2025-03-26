package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

var _ provider.Provider = &ravelinProvider{}

type ravelinProvider struct {
	version string
}

// type ravelinProviderModel struct {
// 	Project types.String `tfsdk:"project"`
// }

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

func (p *ravelinProvider) Configure(_ context.Context, _ provider.ConfigureRequest, _ *provider.ConfigureResponse) {
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
	}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &ravelinProvider{
			version: version,
		}
	}
}
