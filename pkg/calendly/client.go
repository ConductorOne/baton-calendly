package calendly

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/uhttp"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
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

func (c *Client) GetCurrentUser(ctx context.Context) (*User, *v2.RateLimitDescription, error) {
	u := c.prepareURL(fmt.Sprintf(UserEndpoint, "me"))

	var res SingleResponse[User]
	rldata, err := c.get(ctx, u, &res, nil)
	if err != nil {
		return nil, nil, err
	}

	return &res.Resource, rldata, nil
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
	_, err := c.get(ctx, u, &res, queryParams)
	if err != nil {
		return nil, "", err
	}

	return res.Collection, res.Pagination.Next, nil
}

func (c *Client) GetOrgDetails(ctx context.Context, orgURI string) (*Organization, *v2.RateLimitDescription, error) {
	var res SingleResponse[Organization]

	u, err := url.Parse(orgURI)
	if err != nil {
		return nil, nil, err
	}

	rldata, err := c.get(ctx, u, &res, nil)
	if err != nil {
		return nil, nil, err
	}

	return &res.Resource, rldata, nil
}

func (c *Client) RemoveOrgMember(ctx context.Context, membershipID string) (*v2.RateLimitDescription, error) {
	u := c.prepareURL(fmt.Sprintf(OrgMembershipEndpoint, membershipID))

	return c.delete(ctx, u, nil)
}

type InviteBody struct {
	Email string `json:"email"`
}

func (c *Client) InviteOrgMember(ctx context.Context, orgURI string, email string) (*v2.RateLimitDescription, error) {
	path, err := url.JoinPath(orgURI, OrgInvitesEndpoint)
	if err != nil {
		return nil, err
	}

	u, err := url.Parse(path)
	if err != nil {
		return nil, err
	}

	body := &InviteBody{
		Email: email,
	}

	return c.post(ctx, u, body, nil)
}

func (c *Client) ListUserInvitations(ctx context.Context, orgURI string, pgVars *PaginationVars, filterVars *FilterVars) ([]Invitation, string, *v2.RateLimitDescription, error) {
	path, err := url.JoinPath(orgURI, OrgInvitesEndpoint)
	if err != nil {
		return nil, "", nil, err
	}

	u, err := url.Parse(path)
	if err != nil {
		return nil, "", nil, err
	}

	queryParams := &url.Values{}
	c.prepareQuery(queryParams, pgVars)
	queryParams.Set("status", "pending")

	if filterVars != nil {
		queryParams.Set("email", filterVars.Email)
	}

	var res ListResponse[Invitation]
	rldata, err := c.get(ctx, u, &res, queryParams)
	if err != nil {
		return nil, "", nil, err
	}

	return res.Collection, res.Pagination.Next, rldata, nil
}

func (c *Client) RemoveUserInvitation(ctx context.Context, orgURI, invitationID string) (*v2.RateLimitDescription, error) {
	path, err := url.JoinPath(orgURI, OrgInvitesEndpoint, invitationID)
	if err != nil {
		return nil, err
	}

	u, err := url.Parse(path)
	if err != nil {
		return nil, err
	}

	return c.delete(ctx, u, nil)
}

func (c *Client) get(ctx context.Context, urlAddress *url.URL, response interface{}, queryParams *url.Values) (*v2.RateLimitDescription, error) {
	req, err := c.createRequest(ctx, http.MethodGet, urlAddress, nil, queryParams)
	if err != nil {
		return nil, err
	}

	var rldata *v2.RateLimitDescription
	resp, err := c.wrapper.Do(req, uhttp.WithJSONResponse(response), WithErrorResponse(&ErrorResponse{}), WithRatelimitData(rldata))
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	return rldata, nil
}

func (c *Client) delete(ctx context.Context, urlAddress *url.URL, queryParams *url.Values) (*v2.RateLimitDescription, error) {
	req, err := c.createRequest(ctx, http.MethodDelete, urlAddress, nil, queryParams)
	if err != nil {
		return nil, err
	}

	var rldata *v2.RateLimitDescription
	resp, err := c.wrapper.Do(req, WithErrorResponse(&ErrorResponse{}), WithRatelimitData(rldata))
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	return rldata, nil
}

func (c *Client) post(ctx context.Context, urlAddress *url.URL, body interface{}, queryParams *url.Values) (*v2.RateLimitDescription, error) {
	req, err := c.createRequest(ctx, http.MethodPost, urlAddress, body, queryParams)
	if err != nil {
		return nil, err
	}

	var rldata *v2.RateLimitDescription
	resp, err := c.wrapper.Do(req, WithErrorResponse(&ErrorResponse{}), WithRatelimitData(rldata))
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	return rldata, nil
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

func WithRatelimitData(resource *v2.RateLimitDescription) uhttp.DoOption {
	return func(resp *http.Response) error {
		rl, err := extractRateLimitData(resp)
		if err != nil {
			return err
		}

		resource = &v2.RateLimitDescription{
			Limit:     rl.Limit,
			Remaining: rl.Remaining,
			ResetAt:   rl.ResetAt,
		}

		return nil
	}
}

func WithErrorResponse(resource *ErrorResponse) uhttp.DoOption {
	return func(resp *http.Response) error {
		if resp.StatusCode >= 300 {
			// Decode the JSON response body into the ErrorResponse struct
			if err := json.NewDecoder(resp.Body).Decode(&resource); err != nil {
				return status.Error(codes.Unknown, "Request failed with unknown error")
			}

			// Construct a more detailed error message
			errMsg := fmt.Sprintf("Request failed with status %d: %s", resp.StatusCode, resource.Message)

			return status.Error(codes.Unknown, errMsg)
		}

		return nil
	}
}

func extractRateLimitData(resp *http.Response) (*v2.RateLimitDescription, error) {
	if resp == nil {
		return nil, nil
	}

	var l int64
	var err error
	limit := resp.Header.Get("X-Ratelimit-Limit")
	if limit != "" {
		l, err = strconv.ParseInt(limit, 10, 64)
		if err != nil {
			return nil, err
		}
	}

	var r int64
	remaining := resp.Header.Get("X-Ratelimit-Remaining")
	if remaining != "" {
		r, err = strconv.ParseInt(remaining, 10, 64)
		if err != nil {
			return nil, err
		}
	}

	var resetAt time.Time
	reset := resp.Header.Get("X-Ratelimit-Reset")
	if reset != "" {
		r, err := strconv.ParseInt(reset, 10, 64)
		if err != nil {
			return nil, err
		}

		resetAt = time.Now().Add(time.Second * time.Duration(r))
	}

	return &v2.RateLimitDescription{
		Limit:     l,
		Remaining: r,
		ResetAt:   timestamppb.New(resetAt),
	}, nil
}
