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

	OrgUsersEndpoint = "/organization_memberships"
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

func (c *Client) prepareQuery(pgVars *PaginationVars) *url.Values {
	if pgVars == nil {
		return nil
	}

	q := &url.Values{}
	if pgVars.Count > 0 {
		q.Set("count", fmt.Sprintf("%d", pgVars.Count))
	}

	if pgVars.Next != "" {
		q.Set("page_token", pgVars.Next)
	}

	return q
}

func (c *Client) ListUsersUnderOrg(ctx context.Context, orgURI string, pgVars *PaginationVars) ([]OrgMembership, string, error) {
	u := c.prepareURL(OrgUsersEndpoint)
	queryParams := c.prepareQuery(pgVars)
	queryParams.Set("organization", orgURI)

	var res ListResponse[OrgMembership]
	req, err := c.createRequest(ctx, http.MethodGet, u, nil, queryParams)
	if err != nil {
		return nil, "", err
	}

	err = c.get(req, &res)
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

	req, err := c.createRequest(ctx, http.MethodGet, u, nil, nil)
	if err != nil {
		return nil, err
	}

	err = c.get(req, &res)
	if err != nil {
		return nil, err
	}

	return &res.Resource, nil
}

func (c *Client) get(req *http.Request, response interface{}) error {
	resp, err := c.wrapper.Do(req, uhttp.WithJSONResponse(response), WithErrorResponse(&ErrorResponse{}))
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
