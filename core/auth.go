package core

type AuthService interface {
	Verify(string) error
}
