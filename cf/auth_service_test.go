package cf

import (
	"encoding/base64"
	"encoding/json"
	"testing"
)

func TestGetBearer(t *testing.T) {
	var tests = []struct {
		name      string
		auth      string
		expect    string
		expectErr bool
	}{
		{name: "valid bearer", auth: "Bearer abc123token", expect: "abc123token"},
		{name: "bearer with extra parts", auth: "foo Bearer mytoken extra", expect: "mytoken"},
		{name: "no bearer keyword", auth: "Basic abc123", expectErr: true},
		{name: "empty string", auth: "", expectErr: true},
		{name: "bearer at end with no token", auth: "Bearer", expectErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := getBearer(tt.auth)
			if tt.expectErr {
				if err == nil {
					t.Errorf("expected error, got nil (result=%q)", result)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result != tt.expect {
				t.Errorf("got %q, want %q", result, tt.expect)
			}
		})
	}
}

func TestDecodeJWT(t *testing.T) {
	// Build a valid JWT payload with scopes
	claims := JWTClaims{
		Scope: []string{"cloud_controller.admin", "openid"},
	}
	payloadJSON, _ := json.Marshal(claims)
	payload := base64.RawURLEncoding.EncodeToString(payloadJSON)
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256"}`))
	signature := "fakesig"
	token := header + "." + payload + "." + signature

	t.Run("valid token", func(t *testing.T) {
		result, err := DecodeJWT(token)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result.Scope) != 2 {
			t.Fatalf("expected 2 scopes, got %d", len(result.Scope))
		}
		if result.Scope[0] != "cloud_controller.admin" {
			t.Errorf("scope[0]: got %q, want %q", result.Scope[0], "cloud_controller.admin")
		}
		if result.Scope[1] != "openid" {
			t.Errorf("scope[1]: got %q, want %q", result.Scope[1], "openid")
		}
	})

	t.Run("too few parts", func(t *testing.T) {
		_, err := DecodeJWT("onlyonepart")
		if err == nil {
			t.Error("expected error for invalid token format")
		}
	})

	t.Run("invalid base64 payload", func(t *testing.T) {
		_, err := DecodeJWT("header.!!!invalid!!!.sig")
		if err == nil {
			t.Error("expected error for invalid base64")
		}
	})

	t.Run("invalid json payload", func(t *testing.T) {
		badPayload := base64.RawURLEncoding.EncodeToString([]byte(`not json`))
		_, err := DecodeJWT("header." + badPayload + ".sig")
		if err == nil {
			t.Error("expected error for invalid JSON")
		}
	})

	t.Run("empty scopes", func(t *testing.T) {
		emptyClaims := JWTClaims{Scope: []string{}}
		emptyJSON, _ := json.Marshal(emptyClaims)
		emptyPayload := base64.RawURLEncoding.EncodeToString(emptyJSON)
		result, err := DecodeJWT("h." + emptyPayload + ".s")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result.Scope) != 0 {
			t.Errorf("expected 0 scopes, got %d", len(result.Scope))
		}
	})
}
