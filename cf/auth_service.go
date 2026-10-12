package cf

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"strings"

	uaa "github.com/cloudfoundry-community/go-uaa"
	cfclient "github.com/cloudfoundry/go-cfclient/v3/client"
	"github.com/cloudfoundry/go-cfclient/v3/resource"
	"golang.org/x/oauth2"

	"github.com/cloudfoundry-community/ocf-scheduler/core"
)

type Logger interface {
	Info(tag, message string)
	Error(tag, message string)
	Debug(tag, message string)
}

// Identity is the UAA user a bearer token was issued to. UserGUID is the UAA
// user ID, which Cloud Controller uses as the user's GUID.
type Identity struct {
	UserGUID string
	Username string
}

// IdentityFunc resolves a bearer token to its user. Asking UAA also proves
// that the token is valid, which is what lets the scopes decoded from the
// token be trusted afterwards.
type IdentityFunc func(bearer string) (*Identity, error)

type AuthService struct {
	client   CFClient
	logger   Logger
	identify IdentityFunc
}

type caller struct {
	userGUID string
	admin    bool
}

func NewAuthService(client CFClient, logger Logger) *AuthService {
	return NewAuthServiceWithIdentity(client, logger, uaaIdentity)
}

// NewAuthServiceWithIdentity builds an AuthService that resolves tokens with
// the given function instead of UAA's userinfo endpoint.
func NewAuthServiceWithIdentity(client CFClient, logger Logger, identify IdentityFunc) *AuthService {
	return &AuthService{
		client:   client,
		logger:   logger,
		identify: identify,
	}
}

// Verify accepts cloud_controller.admin, or a space_developer or
// space_manager role in any space.
func (service *AuthService) Verify(auth string) error {
	tag := "AuthService.Verify"

	who, err := service.authenticate(auth)
	if err != nil {
		service.logger.Error(tag, err.Error())
		return err
	}

	if who.admin {
		service.logger.Debug(tag, "caller has cloud_controller.admin scope")
		return nil
	}

	opts := cfclient.NewRoleListOptions()
	opts.UserGUIDs.EqualTo(who.userGUID)
	opts.Types.EqualTo("space_developer", "space_manager")

	roles, err := service.client.ListRoles(context.Background(), opts)
	if err != nil {
		service.logger.Error(tag, fmt.Sprintf("error listing roles for user %s: %v", who.userGUID, err))
		return fmt.Errorf("%w: role lookup failed", core.ErrForbidden)
	}

	for _, role := range roles {
		if (role.Type == "space_developer" || role.Type == "space_manager") && roleUser(role) == who.userGUID {
			service.logger.Debug(tag, fmt.Sprintf("user %s has role %s", who.userGUID, role.Type))
			return nil
		}
	}

	service.logger.Error(tag, fmt.Sprintf("user %s has no space role", who.userGUID))
	return fmt.Errorf("%w: insufficient permissions", core.ErrForbidden)
}

// VerifySpace accepts cloud_controller.admin, or a space_developer role in
// spaceGUID. A failed role lookup denies the caller.
func (service *AuthService) VerifySpace(auth string, spaceGUID string) error {
	tag := "AuthService.VerifySpace"

	who, err := service.authenticate(auth)
	if err != nil {
		service.logger.Error(tag, err.Error())
		return err
	}

	if who.admin {
		service.logger.Debug(tag, "caller has cloud_controller.admin scope")
		return nil
	}

	if len(spaceGUID) == 0 {
		service.logger.Error(tag, fmt.Sprintf("user %s asked for no space", who.userGUID))
		return fmt.Errorf("%w: no space given", core.ErrForbidden)
	}

	opts := cfclient.NewRoleListOptions()
	opts.UserGUIDs.EqualTo(who.userGUID)
	opts.SpaceGUIDs.EqualTo(spaceGUID)
	opts.Types.EqualTo("space_developer")

	roles, err := service.client.ListRoles(context.Background(), opts)
	if err != nil {
		service.logger.Error(tag, fmt.Sprintf("error listing roles for user %s in space %s: %v", who.userGUID, spaceGUID, err))
		return fmt.Errorf("%w: role lookup failed", core.ErrForbidden)
	}

	// Match every field ourselves rather than trusting the API's filters.
	for _, role := range roles {
		if role.Type == "space_developer" && roleUser(role) == who.userGUID && roleSpace(role) == spaceGUID {
			service.logger.Debug(tag, fmt.Sprintf("user %s is a space developer in %s", who.userGUID, spaceGUID))
			return nil
		}
	}

	service.logger.Error(tag, fmt.Sprintf("user %s is not a space developer in %s", who.userGUID, spaceGUID))
	return fmt.Errorf("%w: not a space developer in the space", core.ErrForbidden)
}

// IsAdmin reports whether the caller authenticates and holds
// cloud_controller.admin.
func (service *AuthService) IsAdmin(auth string) bool {
	who, err := service.authenticate(auth)
	if err != nil {
		service.logger.Error("AuthService.IsAdmin", err.Error())
		return false
	}

	return who.admin
}

func (service *AuthService) authenticate(auth string) (*caller, error) {
	bearer, err := getBearer(auth)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", core.ErrUnauthenticated, err)
	}

	identity, err := service.identify(bearer)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", core.ErrUnauthenticated, err)
	}

	if identity == nil || len(identity.UserGUID) == 0 {
		return nil, fmt.Errorf("%w: token has no user", core.ErrUnauthenticated)
	}

	scopes, err := getTokenScopes(auth, service.logger)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", core.ErrUnauthenticated, err)
	}

	return &caller{
		userGUID: identity.UserGUID,
		admin:    slices.Contains(scopes, "cloud_controller.admin"),
	}, nil
}

func roleUser(role *resource.Role) string {
	if role.Relationships.User.Data == nil {
		return ""
	}

	return role.Relationships.User.Data.GUID
}

func roleSpace(role *resource.Role) string {
	if role.Relationships.Space.Data == nil {
		return ""
	}

	return role.Relationships.Space.Data.GUID
}

func uaaIdentity(bearer string) (*Identity, error) {
	endpoint := os.Getenv("UAA_ENDPOINT")
	if len(endpoint) == 0 {
		return nil, fmt.Errorf("no UAA endpoint")
	}

	opts := make([]uaa.Option, 0)
	opts = append(opts, uaa.WithSkipSSLValidation(true))

	client, err := uaa.New(endpoint, uaa.WithToken(&oauth2.Token{AccessToken: bearer}), opts...)
	if err != nil {
		return nil, err
	}

	me, err := client.GetMe()
	if err != nil {
		return nil, fmt.Errorf("couldn't get user info: %s", err.Error())
	}

	return &Identity{UserGUID: me.UserID, Username: me.Username}, nil
}

func getBearer(auth string) (string, error) {
	parts := strings.Fields(auth)
	bearerLoc := -1

	// The auth scheme is case-insensitive (RFC 7235), and `cf oauth-token`
	// prints it as "bearer".
	for idx, token := range parts {
		if strings.EqualFold(token, "Bearer") {
			bearerLoc = idx
			break
		}
	}

	if bearerLoc < 0 {
		return "", fmt.Errorf("invalid auth format")
	}

	if len(parts) < bearerLoc+2 {
		return "", fmt.Errorf("invalid bearer token")
	}

	return parts[bearerLoc+1], nil
}

// JWTClaims represents the claims in the JWT token
type JWTClaims struct {
	Scope []string `json:"scope"`
}

// DecodeJWT decodes the JWT token and extracts the claims
func DecodeJWT(token string) (*JWTClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) < 3 {
		return nil, fmt.Errorf("invalid token format")
	}

	payload := parts[1]
	payloadDecoded, err := base64.RawURLEncoding.DecodeString(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to decode payload: %v", err)
	}

	var claims JWTClaims
	err = json.Unmarshal(payloadDecoded, &claims)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal claims: %v", err)
	}

	return &claims, nil
}

func getTokenScopes(auth string, logger Logger) ([]string, error) {
	tag := "AuthService.getTokenScopes"
	bearer, err := getBearer(auth)
	if err != nil {
		logger.Error(tag, fmt.Sprintf("Error getting bearer token: %v", err))
		return nil, err
	}

	claims, err := DecodeJWT(bearer)
	if err != nil {
		logger.Error(tag, fmt.Sprintf("Error decoding JWT token: %v", err))
		return nil, err
	}

	logger.Debug(tag, fmt.Sprintf("Scopes found in token: %v", claims.Scope))
	return claims.Scope, nil
}
