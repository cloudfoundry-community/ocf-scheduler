package cf_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	cfclient "github.com/cloudfoundry/go-cfclient/v3/client"
	"github.com/cloudfoundry/go-cfclient/v3/resource"

	"github.com/cloudfoundry-community/ocf-scheduler/cf"
	"github.com/cloudfoundry-community/ocf-scheduler/core"
	"github.com/cloudfoundry-community/ocf-scheduler/logger"
	"github.com/cloudfoundry-community/ocf-scheduler/mock"
)

// bearer builds an Authorization value whose JWT names user and carries
// scopes. The identity function below reads the user back out of it.
func bearer(user string, scopes ...string) string {
	payload, _ := json.Marshal(map[string]any{"scope": scopes, "user_id": user})
	return "Bearer " + base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none"}`)) + "." +
		base64.RawURLEncoding.EncodeToString(payload) + ".sig"
}

// identify stands in for UAA's userinfo endpoint.
func identify(token string) (*cf.Identity, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid token")
	}
	claims := struct {
		UserID string `json:"user_id"`
	}{}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil || json.Unmarshal(payload, &claims) != nil || claims.UserID == "" {
		return nil, fmt.Errorf("invalid token")
	}
	return &cf.Identity{UserGUID: claims.UserID, Username: claims.UserID}, nil
}

// unfilteredClient answers every role query with the same roles, as a CF API
// that ignored the query's filters would.
type unfilteredClient struct {
	*mock.CFClient
	roles []*resource.Role
}

func (client *unfilteredClient) ListRoles(_ context.Context, _ *cfclient.RoleListOptions) ([]*resource.Role, error) {
	return client.roles, nil
}

func role(roleType, user, space string) *resource.Role {
	return &resource.Role{
		Type: roleType,
		Relationships: resource.RoleSpaceUserOrganizationRelationships{
			User:  resource.ToOneRelationship{Data: &resource.Relationship{GUID: user}},
			Space: resource.ToOneRelationship{Data: &resource.Relationship{GUID: space}},
		},
	}
}

func newService(t *testing.T) (*cf.AuthService, *mock.CFClient) {
	t.Helper()
	client, err := mock.NewCFClient()
	if err != nil {
		t.Fatal(err)
	}
	client.GrantRole("dev", "space-a", "space_developer")
	client.GrantRole("mgr", "space-a", "space_manager")
	client.GrantRole("aud", "space-a", "space_auditor")
	return cf.NewAuthServiceWithIdentity(client, logger.New(), identify), client
}

func TestVerifySpace(t *testing.T) {
	service, _ := newService(t)

	tests := []struct {
		name  string
		auth  string
		space string
		want  error
	}{
		{name: "developer in the space", auth: bearer("dev", "openid"), space: "space-a"},
		{name: "developer in another space", auth: bearer("dev", "openid"), space: "space-b", want: core.ErrForbidden},
		{name: "manager in the space", auth: bearer("mgr", "openid"), space: "space-a", want: core.ErrForbidden},
		{name: "auditor in the space", auth: bearer("aud", "openid"), space: "space-a", want: core.ErrForbidden},
		{name: "user with no roles", auth: bearer("nobody", "openid"), space: "space-a", want: core.ErrForbidden},
		{name: "no space", auth: bearer("dev", "openid"), space: "", want: core.ErrForbidden},
		{name: "admin in any space", auth: bearer("root", "openid", "cloud_controller.admin"), space: "space-b"},
		{name: "admin read-only is not admin", auth: bearer("ro", "openid", "cloud_controller.admin_read_only"), space: "space-a", want: core.ErrForbidden},
		{name: "no bearer", auth: "Basic dXNlcjpwYXNz", space: "space-a", want: core.ErrUnauthenticated},
		{name: "token UAA rejects", auth: "Bearer garbage", space: "space-a", want: core.ErrUnauthenticated},
		{name: "empty auth", auth: "", space: "space-a", want: core.ErrUnauthenticated},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.VerifySpace(tt.auth, tt.space)
			if tt.want == nil && err != nil {
				t.Fatalf("got %v, want access", err)
			}
			if tt.want != nil && !errors.Is(err, tt.want) {
				t.Fatalf("got %v, want %v", err, tt.want)
			}
		})
	}
}

func TestVerifySpaceDeniesWhenRoleLookupFails(t *testing.T) {
	service, client := newService(t)
	client.FailRoleLookups(true)

	if err := service.VerifySpace(bearer("dev", "openid"), "space-a"); !errors.Is(err, core.ErrForbidden) {
		t.Fatalf("got %v, want %v", err, core.ErrForbidden)
	}

	// Admins never need the lookup.
	if err := service.VerifySpace(bearer("root", "cloud_controller.admin"), "space-a"); err != nil {
		t.Fatalf("admin: got %v, want access", err)
	}
}

// A role for another user or another space never counts, even when the CF
// API hands it back for a query that should have filtered it out.
func TestVerifySpaceMatchesRolesStrictly(t *testing.T) {
	base, _ := mock.NewCFClient()
	client := &unfilteredClient{CFClient: base, roles: []*resource.Role{
		role("space_developer", "dev", "space-other"),
		role("space_developer", "someone-else", "space-a"),
		role("space_manager", "dev", "space-a"),
		{Type: "space_developer"},
	}}
	service := cf.NewAuthServiceWithIdentity(client, logger.New(), identify)

	if err := service.VerifySpace(bearer("dev", "openid"), "space-a"); !errors.Is(err, core.ErrForbidden) {
		t.Fatalf("got %v, want %v", err, core.ErrForbidden)
	}

	client.roles = append(client.roles, role("space_developer", "dev", "space-a"))
	if err := service.VerifySpace(bearer("dev", "openid"), "space-a"); err != nil {
		t.Fatalf("got %v, want access", err)
	}
}

func TestVerify(t *testing.T) {
	service, client := newService(t)

	for _, user := range []string{"dev", "mgr"} {
		if err := service.Verify(bearer(user, "openid")); err != nil {
			t.Errorf("%s: got %v, want access", user, err)
		}
	}
	if err := service.Verify(bearer("root", "cloud_controller.admin")); err != nil {
		t.Errorf("admin: got %v, want access", err)
	}
	for _, user := range []string{"aud", "nobody"} {
		if err := service.Verify(bearer(user, "openid")); err == nil {
			t.Errorf("%s: got access, want an error", user)
		}
	}
	if err := service.Verify("Bearer garbage"); !errors.Is(err, core.ErrUnauthenticated) {
		t.Errorf("bad token: got %v, want %v", err, core.ErrUnauthenticated)
	}

	client.FailRoleLookups(true)
	if err := service.Verify(bearer("dev", "openid")); err == nil {
		t.Errorf("failed lookup: got access, want an error")
	}
}
