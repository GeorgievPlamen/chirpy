package auth

import (
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestMakeJWTAndValidateJWT(t *testing.T) {
	userID := uuid.New()
	secret := "super-secret"

	token, err := MakeJWT(userID, secret, time.Hour)
	if err != nil {
		t.Fatalf("expected no error creating token, got %v", err)
	}

	gotUserID, err := ValidateJWT(token, secret)
	if err != nil {
		t.Fatalf("expected no error validating token, got %v", err)
	}

	if gotUserID != userID {
		t.Fatalf("expected user ID %v, got %v", userID, gotUserID)
	}
}

func TestValidateJWTWithWrongSecret(t *testing.T) {
	userID := uuid.New()

	token, err := MakeJWT(userID, "correct-secret", time.Hour)
	if err != nil {
		t.Fatalf("expected no error creating token, got %v", err)
	}

	_, err = ValidateJWT(token, "wrong-secret")
	if err == nil {
		t.Fatal("expected error when validating token with wrong secret")
	}
}

func TestValidateExpiredJWT(t *testing.T) {
	userID := uuid.New()
	secret := "super-secret"

	token, err := MakeJWT(userID, secret, -time.Hour)
	if err != nil {
		t.Fatalf("expected no error creating expired token, got %v", err)
	}

	_, err = ValidateJWT(token, secret)
	if err == nil {
		t.Fatal("expected error when validating expired token")
	}
}

func TestValidateMalformedJWT(t *testing.T) {
	_, err := ValidateJWT("not-a-real-token", "super-secret")
	if err == nil {
		t.Fatal("expected error for malformed token")
	}
}

func TestBearerToken(t *testing.T) {
	cases := []struct {
		key string
		val string
	}{
		{
			key: "Bearer test",
			val: "test",
		},
		{
			key: "test",
			val: "",
		},
	}

	for _, c := range cases {
		header := http.Header{
			"Authorization": []string{
				c.key,
			},
		}
		res, err := GetBearerToken(header)
		if c.val == "" && err == nil {
			t.Fatal("Expected to error for invalid auth bearer schema")
		}

		if res != c.val {
			t.Fatalf("Expected: '%v', Actual: '%v'", res, c.val)
		}
	}
}
