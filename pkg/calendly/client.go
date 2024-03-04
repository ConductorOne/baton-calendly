package calendly

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/conductorone/baton-sdk/pkg/uhttp"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	BaseHost = "api.calendly.com"

	OrgUsersEndpoint      = "/organization_memberships"
	OrgMembershipEndpoint = "/organization_memberships/%s"
	OrgInvitesEndpoint    = "/invitations"

	UserEndpoint = "/users/%s"
)

type Client struct {
	wrapper *uhttp.BaseHttpClient
	baseURL *url.URL
}

func NewClient(httpClient *http.Client) *Client {
	return &Client{
		wrapper: uhttp.NewBaseHttpClient(httpClient),
		baseURL: &url.URL{
			Scheme: "https",
			Host:   BaseHost,
		},
	}
}

type PaginationVars struct {
	Count int    `json:"count"`
	Next  string `json:"next_page_token"`
}

func NewPaginationVars(count int, next string) *PaginationVars {
	return &PaginationVars{
		Count: count,
		Next:  next,
	}
}

type FilterVars struct {
	Email string `json:"email"`
}

func NewFilterVars(email string) *FilterVars {
	return &FilterVars{
		Email: email,
	}
}

type ListResponse[T any] struct {
	Collection []T             `json:"collection"`
	Pagination *PaginationVars `json:"pagination"`
}

type SingleResponse[T any] struct {
	Resource T `json:"resource"`
}

type ErrorResponse struct {
	Title   string `json:"title"`
	Message string `json:"message"`
}

func (c *Client) prepareURL(path string) *url.URL {
	u := *c.baseURL
	u.Path = path
	return &u
}

func (c *Client) prepareQuery(vals *url.Values, pgVars *PaginationVars) {
	if pgVars == nil {
		return
	}

	if pgVars.Count > 0 {
		vals.Set("count", fmt.Sprintf("%d", pgVars.Count))
	}

	if pgVars.Next != "" {
		vals.Set("page_token", pgVars.Next)
	}
}

func (c *Client) GetCurrentUser(ctx context.Context) (*User, error) {
	u := c.prepareURL(fmt.Sprintf(UserEndpoint, "me"))

	var res SingleResponse[User]
	err := c.get(ctx, u, &res, nil)
	if err != nil {
		return nil, err
	}

	return &res.Resource, nil
}

func (c *Client) ListUsersUnderOrg(ctx context.Context, orgURI string, pgVars *PaginationVars, filterVars *FilterVars) ([]OrgMembership, string, error) {
	u := c.prepareURL(OrgUsersEndpoint)
	queryParams := &url.Values{}
	c.prepareQuery(queryParams, pgVars)
	queryParams.Set("organization", orgURI)

	if filterVars != nil {
		queryParams.Set("email", filterVars.Email)
	}

	var res ListResponse[OrgMembership]
	err := c.get(ctx, u, &res, queryParams)
	if err != nil {
		return nil, "", err
	}

	return res.Collection, res.Pagination.Next, nil
}

func (c *Client) GetOrgDetails(ctx context.Context, orgURI string) (*Organization, error) {
	var res SingleResponse[Organization]

	u, err := url.Parse(orgURI)
	if err != nil {
		return nil, err
	}

	err = c.get(ctx, u, &res, nil)
	if err != nil {
		return nil, err
	}

	return &res.Resource, nil
}

func (c *Client) RemoveOrgMember(ctx context.Context, membershipID string) error {
	u := c.prepareURL(fmt.Sprintf(OrgMembershipEndpoint, membershipID))

	return c.delete(ctx, u, nil)
}

type InviteBody struct {
	Email string `json:"email"`
}

func (c *Client) InviteOrgMember(ctx context.Context, orgURI string, email string) error {
	path, err := url.JoinPath(orgURI, OrgInvitesEndpoint)
	if err != nil {
		return err
	}

	u, err := url.Parse(path)
	if err != nil {
		return err
	}

	body := &InviteBody{
		Email: email,
	}

	return c.post(ctx, u, body, nil)
}

func (c *Client) ListUserInvitations(ctx context.Context, orgURI string, pgVars *PaginationVars, filterVars *FilterVars) ([]Invitation, string, error) {
	path, err := url.JoinPath(orgURI, OrgInvitesEndpoint)
	if err != nil {
		return nil, "", err
	}

	u, err := url.Parse(path)
	if err != nil {
		return nil, "", err
	}

	queryParams := &url.Values{}
	c.prepareQuery(queryParams, pgVars)
	queryParams.Set("status", "pending")

	if filterVars != nil {
		queryParams.Set("email", filterVars.Email)
	}

	var res ListResponse[Invitation]
	err = c.get(ctx, u, &res, queryParams)
	if err != nil {
		return nil, "", err
	}

	return res.Collection, res.Pagination.Next, nil
}

func (c *Client) RemoveUserInvitation(ctx context.Context, orgURI, invitationID string) error {
	path, err := url.JoinPath(orgURI, OrgInvitesEndpoint, invitationID)
	if err != nil {
		return err
	}

	u, err := url.Parse(path)
	if err != nil {
		return err
	}

	return c.delete(ctx, u, nil)
}

func (c *Client) get(ctx context.Context, urlAddress *url.URL, response interface{}, queryParams *url.Values) error {
	req, err := c.createRequest(ctx, http.MethodGet, urlAddress, nil, queryParams)
	if err != nil {
		return err
	}

	resp, err := c.wrapper.Do(req, uhttp.WithJSONResponse(response), WithErrorResponse(&ErrorResponse{}))
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	return nil
}

func (c *Client) delete(ctx context.Context, urlAddress *url.URL, queryParams *url.Values) error {
	req, err := c.createRequest(ctx, http.MethodDelete, urlAddress, nil, queryParams)
	if err != nil {
		return err
	}

	resp, err := c.wrapper.Do(req, WithErrorResponse(&ErrorResponse{}))
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	return nil
}

func (c *Client) post(ctx context.Context, urlAddress *url.URL, body interface{}, queryParams *url.Values) error {
	req, err := c.createRequest(ctx, http.MethodPost, urlAddress, body, queryParams)
	if err != nil {
		return err
	}

	resp, err := c.wrapper.Do(req, WithErrorResponse(&ErrorResponse{}))
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	return nil
}

func (c *Client) createRequest(ctx context.Context, method string, urlAddress *url.URL, body interface{}, queryParams *url.Values) (*http.Request, error) {
	req, err := c.wrapper.NewRequest(
		ctx,
		method,
		urlAddress,
		uhttp.WithJSONBody(body),
	)
	if err != nil {
		return nil, err
	}

	if queryParams != nil {
		req.URL.RawQuery = queryParams.Encode()
	}

	return req, nil
}

func WithErrorResponse(resource *ErrorResponse) uhttp.DoOption {
	return func(res *http.Response) error {
		if res.StatusCode >= 300 {
			// Decode the JSON response body into the ErrorResponse struct
			if err := json.NewDecoder(res.Body).Decode(&resource); err != nil {
				return status.Error(codes.Unknown, "Request failed with unknown error")
			}

			// Construct a more detailed error message
			errMsg := fmt.Sprintf("Request failed with status %d: %s", res.StatusCode, resource.Message)

			return status.Error(codes.Unknown, errMsg)
		}

		return nil
	}
}
