package core

import (
	"github.com/ess/dry"
)

func Causify(value dry.Value) string {
	cause := value.(string)

	return cause
}
