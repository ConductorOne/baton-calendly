package connector

import (
	"context"
	"io"
	"net/http"

	"github.com/conductorone/baton-calendly/pkg/calendly"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/connectorbuilder"
	"github.com/conductorone/baton-sdk/pkg/uhttp"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Calendly struct {
	client *calendly.Client
}

// ResourceSyncers returns a ResourceSyncer for each resource type that should be synced from the upstream service.
func (c *Calendly) ResourceSyncers(ctx context.Context) []connectorbuilder.ResourceSyncer {
	return []connectorbuilder.ResourceSyncer{
		newOrgBuilder(c.client),
		newUserBuilder(c.client),
	}
}

// Asset takes an input AssetRef and attempts to fetch it using the connector's authenticated http client
// It streams a response, always starting with a metadata object, following by chunked payloads for the asset.
func (c *Calendly) Asset(ctx context.Context, asset *v2.AssetRef) (string, io.ReadCloser, error) {
	return "", nil, nil
}

// Metadata returns metadata about the connector.
func (c *Calendly) Metadata(ctx context.Context) (*v2.ConnectorMetadata, error) {
	return &v2.ConnectorMetadata{
		DisplayName: "Calendly",
		Description: "Connector syncing Calendly organization and its users and roles to Baton",
	}, nil
}

// Validate is called to ensure that the connector is properly configured. It should exercise any API credentials
// to be sure that they are valid.
func (c *Calendly) Validate(ctx context.Context) (annotations.Annotations, error) {
	u, _, err := c.client.GetCurrentUser(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "calendly-connector: failed to validate credentials")
	}

	_, _, err = c.client.GetOrgDetails(ctx, u.OrgURI)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "calendly-connector: failed to validate credentials")
	}

	return nil, nil
}

// New returns a new instance of the connector.
func New(ctx context.Context, token string) (*Calendly, error) {
	var (
		httpClient *http.Client
		err        error
	)
	if token != "" {
		httpClient, err = uhttp.NewBearerAuth(token).GetClient(ctx)
		if err != nil {
			return nil, err
		}

		return &Calendly{
			client: calendly.NewClient(httpClient),
		}, nil
	}

	auth := &uhttp.NoAuth{}
	httpClient, err = auth.GetClient(ctx)
	if err != nil {
		return nil, err
	}

	return &Calendly{
		client: calendly.NewClient(httpClient),
	}, nil
}
