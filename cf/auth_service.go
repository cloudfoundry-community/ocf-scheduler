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
)

type Logger interface {
	Info(tag, message string)
	Error(tag, message string)
	Debug(tag, message string)
}

type AuthService struct {
	client CFClient
	logger Logger
}

func NewAuthService(client CFClient, logger Logger) *AuthService {
	return &AuthService{
		client: client,
		logger: logger,
	}
}

func (service *AuthService) Verify(auth string) error {
	tag := "AuthService.Verify"
	service.logger.Debug(tag, "Starting verification process")
	username, err := getUsername(auth)
	if err != nil {
		service.logger.Error(tag, fmt.Sprintf("Error getting username: %v", err))
		return err
	}
	service.logger.Debug(tag, fmt.Sprintf("Username obtained: %s", username))

	user, err := service.getUser(username)
	if err != nil {
		service.logger.Error(tag, fmt.Sprintf("Error getting user: %v", err))
		return err
	}

	service.logger.Debug(tag, fmt.Sprintf("User obtained: %v", user))

	roles, err := service.getUserRoles(user)
	if err != nil {
		service.logger.Error(tag, fmt.Sprintf("Error getting user roles: %v", err))
		return err
	}

	service.logger.Debug(tag, fmt.Sprintf("User roles obtained: %v", roles))

	tokenScopes, err := getTokenScopes(auth, service.logger)
	if err != nil {
		service.logger.Error(tag, fmt.Sprintf("Error getting token scopes: %v", err))
		return err
	}
	service.logger.Debug(tag, fmt.Sprintf("Token scopes obtained: %v", tokenScopes))

	// Check all the roles, but return good early if we find one that works.

	// Check token scopes for cloud_controller.admin
	if slices.Contains(tokenScopes, "cloud_controller.admin") {
		service.logger.Debug(tag, "User has cloud_controller.admin scope")
		return nil
	}

	// Check CF roles for space_manager or space_developer
	for _, role := range roles {
		if role.Type == "space_manager" || role.Type == "space_developer" {
			service.logger.Debug(tag, fmt.Sprintf("User has role: %s", role.Type))
			return nil
		}
	}

	service.logger.Error(tag, "User does not have sufficient permissions")
	return fmt.Errorf("insufficient permissions")
}

func (service *AuthService) getUser(username string) (*resource.User, error) {
	tag := "AuthService.getUser"
	opts := cfclient.NewUserListOptions()
	opts.UserNames.EqualTo(username)

	users, err := service.client.ListUsers(context.Background(), opts)
	if err != nil {
		service.logger.Error(tag, fmt.Sprintf("Error listing users: %v", err))
		return nil, err
	}

	if len(users) == 0 {
		service.logger.Error(tag, "No such user found")
		return nil, fmt.Errorf("no such user")
	}

	return users[0], nil
}

func (service *AuthService) getUserRoles(user *resource.User) ([]*resource.Role, error) {
	tag := "AuthService.getUserRoles"
	opts := cfclient.NewRoleListOptions()
	opts.UserGUIDs.EqualTo(user.GUID)

	roles, err := service.client.ListRoles(context.Background(), opts)
	if err != nil {
		service.logger.Error(tag, fmt.Sprintf("Error listing roles: %v", err))
		return nil, err
	}

	return roles, nil
}

func getUsername(auth string) (string, error) {
	endpoint := os.Getenv("UAA_ENDPOINT")
	if len(endpoint) == 0 {
		return "", fmt.Errorf("no UAA endpoint")
	}

	bearer, err := getBearer(auth)
	if err != nil {
		return "", err
	}

	opts := make([]uaa.Option, 0)
	opts = append(opts, uaa.WithSkipSSLValidation(true))

	client, err := uaa.New(endpoint, uaa.WithToken(&oauth2.Token{AccessToken: bearer}), opts...)
	if err != nil {
		return "", err
	}

	me, err := client.GetMe()
	if err != nil {
		return "", fmt.Errorf("couldn't get user info: %s", err.Error())
	}

	return me.Username, nil
}

func getBearer(auth string) (string, error) {
	parts := strings.Split(auth, " ")
	bearerLoc := -1

	for idx, token := range parts {
		if token == "Bearer" {
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
