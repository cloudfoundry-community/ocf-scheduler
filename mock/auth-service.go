package mock

import (
	"fmt"
)

type AuthService struct {
}

func NewAuthService() *AuthService {
	return &AuthService{}
}

func (service *AuthService) Verify(auth string) error {
	if len(auth) == 0 {
		return fmt.Errorf("no auth provided")
	}

	if auth != "jeremy" {
		return fmt.Errorf("you're not jeremy!")
	}

	return nil
}
