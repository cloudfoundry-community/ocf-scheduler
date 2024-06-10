package cf

import (
	"fmt"
	"net/url"
	"os"
	"strings"

	cf "github.com/cloudfoundry-community/go-cfclient"
	uaa "github.com/cloudfoundry-community/go-uaa"
	"golang.org/x/oauth2"
)

type Logger interface {
	Info(tag, message string)
	Error(tag, message string)
}

type AuthService struct {
	client *cf.Client
	logger Logger
}

func NewAuthService(client *cf.Client, logger Logger) *AuthService {
	return &AuthService{
		client: client,
		logger: logger,
	}
}

func (service *AuthService) Verify(auth string) error {
	tag := "AuthService.Verify"
	service.logger.Info(tag, "Starting verification process")
	username, err := getUsername(auth)
	if err != nil {
		service.logger.Error(tag, fmt.Sprintf("Error getting username: %v", err))
		return err
	}
	service.logger.Info(tag, fmt.Sprintf("Username obtained: %s", username))

	user, err := service.getUser(username)
	if err != nil {
		service.logger.Error(tag, fmt.Sprintf("Error getting user: %v", err))
		return err
	}
	//  Debugging = noisy
	//	service.logger.Info(tag, fmt.Sprintf("User obtained: %v", user))

	roles, err := service.getUserRoles(user)
	if err != nil {
		service.logger.Error(tag, fmt.Sprintf("Error getting user roles: %v", err))
		return err
	}
	//  Debugging = noisy
	// service.logger.Info(tag, fmt.Sprintf("User roles obtained: %v", roles))

	tokenScopes, err := getTokenScopes(auth, service.logger)
	if err != nil {
		service.logger.Error(tag, fmt.Sprintf("Error getting token scopes: %v", err))
		return err
	}
	service.logger.Info(tag, fmt.Sprintf("Token scopes obtained: %v", tokenScopes))

	// Check all the roles, but return good early if we find one that works.
	// Check CF roles for space_manager or space_developer
	for _, role := range roles {
		// NOTE: we should definitely be checking space IDs, too, but that's tomorrow
		// guy's problem.
		if role.Type == "space_manager" || role.Type == "space_developer" {
			service.logger.Info(tag, fmt.Sprintf("User has role: %s", role.Type))
			return nil
		}
	}

	service.logger.Error(tag, "User does not have sufficient permissions")
	return fmt.Errorf("insufficient permissions")
}

func (service *AuthService) getUser(username string) (cf.User, error) {
	tag := "AuthService.getUser"
	query := url.Values{}
	query.Add("username", username)

	users, err := service.client.ListUsersByQuery(query)
	if err != nil {
		service.logger.Error(tag, fmt.Sprintf("Error listing users by query: %v", err))
		return cf.User{}, err
	}

	user := users.GetUserByUsername(username)
	if len(user.Guid) == 0 {
		service.logger.Error(tag, "No such user found")
		return cf.User{}, fmt.Errorf("no such user")
	}

	return user, nil
}

func (service *AuthService) getUserRoles(user cf.User) ([]cf.V3Role, error) {
	tag := "AuthService.getUserRoles"
	roleQuery := url.Values{}
	roleQuery.Add("user_guids", user.Guid)
	roles, err := service.client.ListV3RolesByQuery(roleQuery)
	if err != nil {
		service.logger.Error(tag, fmt.Sprintf("Error listing V3 roles by query: %v", err))
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
