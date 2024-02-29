package connector

import (
	"context"
	"fmt"
	"time"

	"github.com/conductorone/baton-calendly/pkg/calendly"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/helpers"
	"github.com/conductorone/baton-sdk/pkg/pagination"
	rs "github.com/conductorone/baton-sdk/pkg/types/resource"
)

type userBuilder struct {
	client       *calendly.Client
	resourceType *v2.ResourceType
}

func userResource(user *calendly.User, parentId *v2.ResourceId) (*v2.Resource, error) {
	firstName, lastName := helpers.SplitFullName(user.FullName)
	profile := map[string]interface{}{
		"user_id":   user.ID,
		"email":     user.Email,
		"firstName": firstName,
		"lastName":  lastName,
		"slug":      user.Slug,
	}

	created, err := time.Parse(time.RFC3339, user.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to parse user created_at: %w", err)
	}

	userOptions := []rs.UserTraitOption{
		rs.WithUserProfile(profile),
		rs.WithEmail(user.Email, true),
		rs.WithUserLogin(user.Email),
		rs.WithCreatedAt(created),
		rs.WithStatus(v2.UserTrait_Status_STATUS_ENABLED),
	}

	resource, err := rs.NewUserResource(
		user.Slug,
		userResourceType,
		user.ID,
		userOptions,
		rs.WithParentResourceID(parentId),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create user resource: %w", err)
	}

	return resource, nil
}

func (o *userBuilder) ResourceType(ctx context.Context) *v2.ResourceType {
	return userResourceType
}

// List returns all the users from the database as resource objects.
// Users include a UserTrait because they are the 'shape' of a standard user.
func (o *userBuilder) List(ctx context.Context, parentResourceID *v2.ResourceId, pToken *pagination.Token) ([]*v2.Resource, string, annotations.Annotations, error) {
	if parentResourceID == nil {
		return nil, "", nil, nil
	}

	bag, page, err := parsePageToken(pToken.Token, &v2.ResourceId{ResourceType: userResourceType.Id})
	if err != nil {
		return nil, "", nil, err
	}

	pgVars := calendly.NewPaginationVars(ResourcesPageSize, page)
	users, nextPage, err := o.client.ListUsersUnderOrg(ctx, parentResourceID.Resource, pgVars)
	if err != nil {
		return nil, "", nil, fmt.Errorf("calendly-connector: failed to list users: %w", err)
	}

	var rv []*v2.Resource
	for _, u := range users {
		ur, err := userResource(u.User, parentResourceID)
		if err != nil {
			return nil, "", nil, fmt.Errorf("calendly-connector: failed to create user resource: %w", err)
		}

		rv = append(rv, ur)
	}

	next, err := bag.NextToken(nextPage)
	if err != nil {
		return nil, "", nil, err
	}

	return rv, next, nil, nil
}

// Entitlements always returns an empty slice for users.
func (o *userBuilder) Entitlements(_ context.Context, resource *v2.Resource, _ *pagination.Token) ([]*v2.Entitlement, string, annotations.Annotations, error) {
	return nil, "", nil, nil
}

// Grants always returns an empty slice for users since they don't have any entitlements.
func (o *userBuilder) Grants(ctx context.Context, resource *v2.Resource, pToken *pagination.Token) ([]*v2.Grant, string, annotations.Annotations, error) {
	return nil, "", nil, nil
}

func newUserBuilder(client *calendly.Client) *userBuilder {
	return &userBuilder{
		client:       client,
		resourceType: userResourceType,
	}
}
