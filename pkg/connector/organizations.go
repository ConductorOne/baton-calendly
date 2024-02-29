package connector

import (
	"context"
	"fmt"

	"github.com/conductorone/baton-calendly/pkg/calendly"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/pagination"
	ent "github.com/conductorone/baton-sdk/pkg/types/entitlement"
	rs "github.com/conductorone/baton-sdk/pkg/types/resource"
)

const (
	OrgUserEntitlement  = "user"
	OrgAdminEntitlement = "admin"
	OrgOwnerEntitlement = "owner"
)

var OrgRoles = []string{
	OrgUserEntitlement,
	OrgAdminEntitlement,
	OrgOwnerEntitlement,
}

type orgBuilder struct {
	client *calendly.Client
	URI    string
}

func (o *orgBuilder) ResourceType(ctx context.Context) *v2.ResourceType {
	return orgResourceType
}

func orgResource(org *calendly.Organization) (*v2.Resource, error) {
	id := parseResourceID(org.ID)

	resource, err := rs.NewResource(
		id,
		orgResourceType,
		org.ID,
		rs.WithAnnotation(
			&v2.ChildResourceType{ResourceTypeId: userResourceType.Id},
		),
	)
	if err != nil {
		return nil, err
	}

	return resource, nil
}

// List returns top level resource - the organization from the database.
func (o *orgBuilder) List(ctx context.Context, parentResourceID *v2.ResourceId, pToken *pagination.Token) ([]*v2.Resource, string, annotations.Annotations, error) {
	var rv []*v2.Resource

	orgDetails, err := o.client.GetOrgDetails(ctx, o.URI)
	if err != nil {
		return nil, "", nil, fmt.Errorf("snyk-connector: failed to get org details: %w", err)
	}

	or, err := orgResource(orgDetails)
	if err != nil {
		return nil, "", nil, fmt.Errorf("snyk-connector: failed to create org resource: %w", err)
	}

	rv = append(rv, or)

	return rv, "", nil, nil
}

// Entitlements returns slice of membership and permission entitlements for the org.
func (o *orgBuilder) Entitlements(ctx context.Context, resource *v2.Resource, _ *pagination.Token) ([]*v2.Entitlement, string, annotations.Annotations, error) {
	var rv []*v2.Entitlement

	for _, role := range OrgRoles {
		permissionOptions := []ent.EntitlementOption{
			ent.WithGrantableTo(userResourceType),
			ent.WithDisplayName(fmt.Sprintf("%s role", role)),
			ent.WithDescription(fmt.Sprintf("%s role in the organization", role)),
		}

		rv = append(rv, ent.NewPermissionEntitlement(resource, role, permissionOptions...))
	}

	return rv, "", nil, nil
}

// Grants returns slice of membership and permission grants for the org.
func (o *orgBuilder) Grants(ctx context.Context, resource *v2.Resource, pToken *pagination.Token) ([]*v2.Grant, string, annotations.Annotations, error) {
	// TODO: implement

	return nil, "", nil, nil
}

func (o *orgBuilder) Grant(ctx context.Context, principal *v2.Resource, entitlement *v2.Entitlement) (annotations.Annotations, error) {
	// TODO: implement

	return nil, nil
}

func (o *orgBuilder) Revoke(ctx context.Context, grant *v2.Grant) (annotations.Annotations, error) {
	// TODO: implement

	return nil, nil
}

func newOrgBuilder(client *calendly.Client, uri string) *orgBuilder {
	return &orgBuilder{
		client: client,
		URI:    uri,
	}
}
