package core

import "errors"

var (
	// ErrUnauthenticated means the caller's token could not be verified.
	ErrUnauthenticated = errors.New("unauthenticated")

	// ErrForbidden means the caller is authenticated but may not act in the
	// space, including when their roles could not be looked up.
	ErrForbidden = errors.New("forbidden")
)

type AuthService interface {
	// Verify authenticates the caller and accepts cloud_controller.admin or
	// a space_developer or space_manager role in any space. It guards only
	// routes that touch no tenant data.
	Verify(auth string) error

	// VerifySpace authenticates the caller and accepts cloud_controller.admin
	// or a space_developer role in the given space. Errors wrap
	// ErrUnauthenticated or ErrForbidden.
	VerifySpace(auth string, spaceGUID string) error
}
