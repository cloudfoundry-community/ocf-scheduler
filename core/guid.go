package core

import (
	"fmt"

	"github.com/google/uuid"
)

func GenGUID() (string, error) {
	id, err := uuid.NewRandom()

	if err != nil {
		return "", fmt.Errorf("could not generate a UUID")
	}

	return id.String(), nil
}
