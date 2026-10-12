package broker

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// MinPasswordLength is the shortest broker password the scheduler accepts.
// The broker answers on the scheduler's public route and nothing limits the
// rate of guesses, so the password is its main defence.
const MinPasswordLength = 16

// The environment variables that configure the broker. The BOSH release sets
// them from the scheduler.broker job properties.
const (
	EnvEnabled        = "SCHEDULER_BROKER_ENABLED"
	EnvUsername       = "SCHEDULER_BROKER_USERNAME"
	EnvPassword       = "SCHEDULER_BROKER_PASSWORD"
	EnvAPIURL         = "SCHEDULER_BROKER_API_URL"
	EnvRequireBinding = "SCHEDULER_BROKER_REQUIRE_BINDING"
)

// Config holds the broker settings. A zero Config means the broker is off.
type Config struct {
	Enabled        bool
	Username       string
	Password       string
	APIURL         string
	RequireBinding bool
}

// ConfigFromEnv reads the broker settings through getenv, which is
// os.Getenv in production. The broker is on only when SCHEDULER_BROKER_ENABLED
// is true, and then the username, password, and API URL must all be set.
// Errors never carry the password.
func ConfigFromEnv(getenv func(string) string) (Config, error) {
	enabled, err := parseBool(EnvEnabled, getenv(EnvEnabled))
	if err != nil {
		return Config{}, err
	}

	requireBinding, err := parseBool(EnvRequireBinding, getenv(EnvRequireBinding))
	if err != nil {
		return Config{}, err
	}

	if !enabled {
		if requireBinding {
			return Config{}, fmt.Errorf("%s is true but the broker is off, so no app could ever be bound; set %s to true or turn %s off", EnvRequireBinding, EnvEnabled, EnvRequireBinding)
		}

		return Config{}, nil
	}

	config := Config{
		Enabled:        true,
		Username:       getenv(EnvUsername),
		Password:       getenv(EnvPassword),
		RequireBinding: requireBinding,
	}

	if len(config.Username) == 0 {
		return Config{}, fmt.Errorf("%s is true but %s is not set", EnvEnabled, EnvUsername)
	}

	// Cloud Controller sends the username exactly as it was registered, so
	// trimming it here would only hide a mismatch until the first request.
	if config.Username != strings.TrimSpace(config.Username) {
		return Config{}, fmt.Errorf("%s has leading or trailing whitespace", EnvUsername)
	}

	if len(config.Password) == 0 {
		return Config{}, fmt.Errorf("%s is true but %s is not set", EnvEnabled, EnvPassword)
	}

	if len(config.Password) < MinPasswordLength {
		return Config{}, fmt.Errorf("%s must be at least %d characters long", EnvPassword, MinPasswordLength)
	}

	apiURL := strings.TrimRight(strings.TrimSpace(getenv(EnvAPIURL)), "/")
	if len(apiURL) == 0 {
		return Config{}, fmt.Errorf("%s is true but %s is not set", EnvEnabled, EnvAPIURL)
	}

	parsed, err := url.Parse(apiURL)
	if err != nil || (parsed.Scheme != "https" && parsed.Scheme != "http") || len(parsed.Host) == 0 {
		return Config{}, fmt.Errorf("%s must be an absolute http or https URL, got %q", EnvAPIURL, apiURL)
	}

	config.APIURL = apiURL

	return config, nil
}

// parseBool reads an optional boolean, where empty means false.
func parseBool(name, raw string) (bool, error) {
	raw = strings.TrimSpace(raw)
	if len(raw) == 0 {
		return false, nil
	}

	value, err := strconv.ParseBool(raw)
	if err != nil {
		return false, fmt.Errorf("%s must be true or false, got %q", name, raw)
	}

	return value, nil
}
