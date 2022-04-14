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

type AuthService struct {
	client *cf.Client
}

func NewAuthService(client *cf.Client) *AuthService {
	return &AuthService{
		client: client,
	}
}

func (service *AuthService) Verify(auth string) error {
	username, err := getUsername(auth)
	if err != nil {
		return err
	}

	user, err := service.getUser(username)
	if err != nil {
		return err
	}

	roles, err := service.getUserRoles(user)
	if err != nil {
		return err
	}

	// Check all the roles, but return good early if we find one that works.
	for _, role := range roles {
		// NOTE: we should definitely be checking space IDs, too, but that's tomorrow
		// guy's problem.
		if role.Type == "space_manager" || role.Type == "space_developer" {
			return nil
		}
	}

	return fmt.Errorf("insufficient permissions")
}

func (service *AuthService) getUser(username string) (cf.User, error) {
	query := url.Values{}
	query.Add("username", username)

	users, err := service.client.ListUsersByQuery(query)
	if err != nil {
		return cf.User{}, err
	}

	user := users.GetUserByUsername(username)
	if len(user.Guid) == 0 {
		return cf.User{}, fmt.Errorf("no such user")
	}

	return user, nil
}

func (service *AuthService) getUserRoles(user cf.User) ([]cf.V3Role, error) {
	roleQuery := url.Values{}
	roleQuery.Add("user_guids", user.Guid)
	roles, err := service.client.ListV3RolesByQuery(roleQuery)
	if err != nil {
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
