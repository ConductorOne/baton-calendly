package calendly

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/conductorone/baton-sdk/pkg/uhttp"
)

// testClientServing returns a client pointed at a stub Calendly that always replies with
// body. Each test gets its own server, so the uhttp response cache cannot carry a body
// between them.
func testClientServing(t *testing.T, body string) (*Client, string) {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)

	client, err := NewClient(server.Client(), server.URL)
	if err != nil {
		t.Fatalf("Expected no error building the client, got %v", err)
	}

	return client, server.URL
}

// Calendly keeps the pagination object on the last page and nulls next_page_token. That
// is the end of the sync, not a failure.
func TestListUsersUnderOrgLastPage(t *testing.T) {
	client, orgURI := testClientServing(t, `{"collection":[{"uri":"m1","role":"user"}],"pagination":{"count":1,"next_page_token":null}}`)

	memberships, next, err := client.ListUsersUnderOrg(context.Background(), orgURI, nil, nil)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if len(memberships) != 1 {
		t.Fatalf("Expected 1 membership, got %d", len(memberships))
	}
	if next != "" {
		t.Errorf("Expected an empty page token on the last page, got %q", next)
	}
}

func TestListUsersUnderOrgNextPageToken(t *testing.T) {
	client, orgURI := testClientServing(t, `{"collection":[{"uri":"m1"}],"pagination":{"count":1,"next_page_token":"tok"}}`)

	_, next, err := client.ListUsersUnderOrg(context.Background(), orgURI, nil, nil)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if next != "tok" {
		t.Errorf("Expected page token %q, got %q", "tok", next)
	}
}

// The case the opt-in exists for. A 200 carrying a collection but no pagination object
// used to read a nil Pagination straight through to Pagination.Next.
func TestListUsersUnderOrgMissingPagination(t *testing.T) {
	client, orgURI := testClientServing(t, `{"collection":[{"uri":"m1"}]}`)

	_, _, err := client.ListUsersUnderOrg(context.Background(), orgURI, nil, nil)
	if err == nil {
		t.Fatal("Expected an error when Calendly omits pagination")
	}
	if !errors.Is(err, uhttp.ErrMissingPaginationData) {
		t.Fatalf("Expected ErrMissingPaginationData, got %v", err)
	}
}

func TestListUserInvitationsMissingPagination(t *testing.T) {
	client, orgURI := testClientServing(t, `{"collection":[{"uri":"i1","email":"a@b.com"}]}`)

	_, _, _, err := client.ListUserInvitations(context.Background(), orgURI, nil, nil)
	if err == nil {
		t.Fatal("Expected an error when Calendly omits pagination")
	}
	if !errors.Is(err, uhttp.ErrMissingPaginationData) {
		t.Fatalf("Expected ErrMissingPaginationData, got %v", err)
	}
}

// Single-resource reads share the same get() and carry no pagination object, so they must
// not start demanding one.
func TestGetOrgDetailsDoesNotRequirePagination(t *testing.T) {
	client, orgURI := testClientServing(t, `{"resource":{"uri":"o1","plan":"standard"}}`)

	org, _, err := client.GetOrgDetails(context.Background(), orgURI)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if org.ID != "o1" {
		t.Errorf("Expected org id %q, got %q", "o1", org.ID)
	}
}
