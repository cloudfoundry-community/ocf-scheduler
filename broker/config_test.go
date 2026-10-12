package broker

import (
	"strings"
	"testing"
)

func env(values map[string]string) func(string) string {
	return func(name string) string { return values[name] }
}

func TestConfigFromEnv(t *testing.T) {
	complete := map[string]string{
		EnvEnabled:  "true",
		EnvUsername: "broker",
		EnvPassword: "secret-password",
		EnvAPIURL:   "https://scheduler.example.com/",
	}

	with := func(changes map[string]string) map[string]string {
		values := map[string]string{}
		for name, value := range complete {
			values[name] = value
		}
		for name, value := range changes {
			values[name] = value
		}
		return values
	}

	t.Run("off when nothing is set", func(t *testing.T) {
		config, err := ConfigFromEnv(env(nil))
		if err != nil || config.Enabled {
			t.Fatalf("want the broker off, got %+v, %v", config, err)
		}
	})

	t.Run("off when disabled even with credentials", func(t *testing.T) {
		config, err := ConfigFromEnv(env(with(map[string]string{EnvEnabled: "false"})))
		if err != nil || config.Enabled || config.Password != "" {
			t.Fatalf("want an empty config, got enabled=%t err=%v", config.Enabled, err)
		}
	})

	t.Run("on with every setting", func(t *testing.T) {
		config, err := ConfigFromEnv(env(with(map[string]string{EnvRequireBinding: "true"})))
		if err != nil {
			t.Fatal(err)
		}
		want := Config{Enabled: true, Username: "broker", Password: "secret-password", APIURL: "https://scheduler.example.com", RequireBinding: true}
		if config != want {
			t.Errorf("got %+v, want %+v", config, want)
		}
	})

	for name, values := range map[string]map[string]string{
		"no username":                        with(map[string]string{EnvUsername: ""}),
		"no password":                        with(map[string]string{EnvPassword: ""}),
		"no api url":                         with(map[string]string{EnvAPIURL: ""}),
		"a relative api url":                 with(map[string]string{EnvAPIURL: "scheduler.example.com"}),
		"an api url with another scheme":     with(map[string]string{EnvAPIURL: "ftp://scheduler.example.com"}),
		"an unreadable enabled":              with(map[string]string{EnvEnabled: "yes please"}),
		"an unreadable require_binding":      with(map[string]string{EnvRequireBinding: "maybe"}),
		"require_binding without the broker": {EnvRequireBinding: "true"},
	} {
		t.Run("refuses "+name, func(t *testing.T) {
			config, err := ConfigFromEnv(env(values))
			if err == nil {
				t.Fatalf("want an error, got %+v", config)
			}
			if strings.Contains(err.Error(), "secret-password") {
				t.Errorf("the error carries the password: %v", err)
			}
		})
	}
}
