package api

import (
	"encoding/json"
	"testing"
)

func TestIdentityUserStatusAsString(t *testing.T) {
	raw := `{
		"id": 1,
		"email": "test@example.com",
		"display_name": "Test",
		"status": "active",
		"locale": "zh-CN",
		"email_verified": true,
		"password_setup_required": false
	}`
	var user IdentityUser
	if err := json.Unmarshal([]byte(raw), &user); err != nil {
		t.Fatalf("unmarshal identity user: %v", err)
	}
	if user.Status != "active" {
		t.Fatalf("status = %q, want active", user.Status)
	}
}
