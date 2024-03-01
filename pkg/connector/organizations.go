package connector

import (
	"context"
	"fmt"
	"slices"

	"github.com/conductorone/baton-calendly/pkg/calendly"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/pagination"
	ent "github.com/conductorone/baton-sdk/pkg/types/entitlement"
	"github.com/conductorone/baton-sdk/pkg/types/grant"
	rs "github.com/conductorone/baton-sdk/pkg/types/resource"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	OrgUserEntitlement        = "user"
	OrgAdminEntitlement       = "admin"
	OrgOwnerEntitlement       = "owner"
	OrgPendingUserEntitlement = "pending_user"

	InvitationsType = "invitations"
)

// TODO: add support for more roles, more information about all roles:
// https://help.calendly.com/hc/en-us/articles/4410722852759-User-roles-and-permissions
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
// TODO: add support for listing multiple organizations.
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

	// entitlement representing invitation to the organization
	iaOptions := []ent.EntitlementOption{
		ent.WithGrantableTo(userResourceType),
		ent.WithDisplayName("pending invitation"),
		ent.WithDescription("pending invitation to the organization"),
	}
	rv = append(rv, ent.NewAssignmentEntitlement(resource, OrgPendingUserEntitlement, iaOptions...))

	// entitlements representing roles in the organization
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
	var rv []*v2.Grant

	bag, page, err := parsePageToken(pToken.Token, resource.Id)
	if err != nil {
		return nil, "", nil, fmt.Errorf("snyk-connector: failed to parse page token: %w", err)
	}

	switch bag.ResourceTypeID() {
	case resource.Id.ResourceType:
		bag.Pop()
		bag.Push(pagination.PageState{
			ResourceTypeID: userResourceType.Id,
		})
		bag.Push(pagination.PageState{
			ResourceTypeID: InvitationsType,
		})

	case InvitationsType:
		pgVars := calendly.NewPaginationVars(ResourcesPageSize, page)
		invitations, nextPage, err := o.client.ListUserInvitations(ctx, o.URI, pgVars, nil)
		if err != nil {
			return nil, "", nil, fmt.Errorf("snyk-connector: failed to list org invitations: %w", err)
		}

		err = bag.Next(nextPage)
		if err != nil {
			return nil, "", nil, err
		}

		for _, i := range invitations {
			userId, err := rs.NewResourceID(userResourceType, i.Email)
			if err != nil {
				return nil, "", nil, fmt.Errorf("snyk-connector: failed to create user resource id: %w", err)
			}

			rv = append(rv, grant.NewGrant(resource, OrgPendingUserEntitlement, userId))
		}

	case userResourceType.Id:
		pgVars := calendly.NewPaginationVars(ResourcesPageSize, page)
		memberships, nextPage, err := o.client.ListUsersUnderOrg(ctx, resource.Id.Resource, pgVars, nil)
		if err != nil {
			return nil, "", nil, fmt.Errorf("snyk-connector: failed to list users in org: %w", err)
		}

		err = bag.Next(nextPage)
		if err != nil {
			return nil, "", nil, err
		}

		for _, m := range memberships {
			// check for valid role
			if !slices.Contains(OrgRoles, m.Role) {
				return nil, "", nil, fmt.Errorf("snyk-connector: role %s not found", m.Role)
			}

			userId, err := rs.NewResourceID(userResourceType, m.User.ID)
			if err != nil {
				return nil, "", nil, fmt.Errorf("snyk-connector: failed to create user resource id: %w", err)
			}

			rv = append(rv, grant.NewGrant(resource, m.Role, userId))
		}

	default:
		return nil, "", nil, fmt.Errorf("snyk-connector: invalid page token")
	}

	next, err := bag.Marshal()
	if err != nil {
		return nil, "", nil, err
	}

	return rv, next, nil, nil
}

// Grant method is only used for user invitations to the organization.
func (o *orgBuilder) Grant(ctx context.Context, principal *v2.Resource, entitlement *v2.Entitlement) (annotations.Annotations, error) {
	l := ctxzap.Extract(ctx)

	// check for principal type
	if principal.Id.ResourceType != userResourceType.Id {
		l.Warn(
			"calendly-connector: only users can be granted organization membership",
			zap.String("principal_id", principal.Id.Resource),
			zap.String("principal_type", principal.Id.ResourceType),
		)

		return nil, status.Error(codes.InvalidArgument, "calendly-connector: only users can be granted organization membership")
	}

	// check for valid role - we can't grant roles - only invitations are allowed
	if entitlement.Slug != OrgPendingUserEntitlement {
		l.Warn(
			"calendly-connector: only user role can be granted in organization",
			zap.String("entitlement_slug", entitlement.Slug),
		)

		return nil, status.Error(codes.InvalidArgument, "calendly-connector: only user role can be granted in organization")
	}

	err := o.client.InviteOrgMember(ctx, o.URI, principal.Id.Resource)
	if err != nil {
		return nil, fmt.Errorf("snyk-connector: failed to invite user to org: %w", err)
	}

	return nil, nil
}

// Revoke method is only used for canceling invitations and removing users from the organization.
// Note that we can't remove owner from the organization.
func (o *orgBuilder) Revoke(ctx context.Context, grant *v2.Grant) (annotations.Annotations, error) {
	l := ctxzap.Extract(ctx)

	entitlement := grant.Entitlement
	principal := grant.Principal

	// check for principal type
	if principal.Id.ResourceType != userResourceType.Id {
		l.Warn(
			"calendly-connector: only users can be revoked from organization",
			zap.String("principal_id", principal.Id.Resource),
			zap.String("principal_type", principal.Id.ResourceType),
		)

		return nil, status.Error(codes.InvalidArgument, "calendly-connector: only users can be revoked from organization")
	}

	// check for valid role - we can't revoke roles - only invitations are allowed
	if entitlement.Slug != OrgUserEntitlement && entitlement.Slug != OrgPendingUserEntitlement {
		l.Warn(
			"calendly-connector: only user memberships and invitations can be revoked in organization",
			zap.String("principal_id", principal.Id.Resource),
			zap.String("principal_type", principal.Id.ResourceType),
		)

		return nil, status.Error(codes.InvalidArgument, "calendly-connector: only user memberships and invitations can be revoked in organization")
	}

	if entitlement.Slug == OrgUserEntitlement {
		memberships, _, err := o.client.ListUsersUnderOrg(ctx, o.URI, nil, calendly.NewFilterVars(principal.DisplayName))
		if err != nil {
			return nil, fmt.Errorf("snyk-connector: failed to list users in org: %w", err)
		}

		if len(memberships) == 0 {
			return nil, status.Error(codes.NotFound, "calendly-connector: user not found in org")
		}

		if len(memberships) > 1 {
			return nil, status.Error(codes.Internal, "calendly-connector: multiple users found in org")
		}

		membershipURI := memberships[0].ID
		membershipID := parseResourceID(membershipURI)
		err = o.client.RemoveOrgMember(ctx, membershipID)
		if err != nil {
			return nil, fmt.Errorf("snyk-connector: failed to remove user from org: %w", err)
		}

		return nil, nil
	}

	if entitlement.Slug == OrgPendingUserEntitlement {
		invitations, _, err := o.client.ListUserInvitations(ctx, o.URI, nil, calendly.NewFilterVars(principal.DisplayName))
		if err != nil {
			return nil, fmt.Errorf("snyk-connector: failed to list org invitations: %w", err)
		}

		if len(invitations) == 0 {
			return nil, status.Error(codes.NotFound, "calendly-connector: user invitation not found in org")
		}

		// do not throw error in case of duplicates, just take the first one
		invitationURI := invitations[0].ID
		invitationID := parseResourceID(invitationURI)
		err = o.client.RemoveUserInvitation(ctx, o.URI, invitationID)
		if err != nil {
			return nil, fmt.Errorf("snyk-connector: failed to remove user invitation: %w", err)
		}

		return nil, nil
	}

	return nil, nil
}

func newOrgBuilder(client *calendly.Client, uri string) *orgBuilder {
	return &orgBuilder{
		client: client,
		URI:    uri,
	}
}
