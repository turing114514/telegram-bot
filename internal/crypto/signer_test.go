package crypto

import (
	"strconv"
	"testing"
	"time"
)

func TestSignVerifyRoundtrip(t *testing.T) {
	secret := "abc123"
	method := "POST"
	path := "/api/v1/channel/telegram/config"
	ts := time.Now().Unix()
	body := []byte(`{"k":"v"}`)

	sig := Sign(secret, method, path, ts, body)
	if sig == "" {
		t.Fatal("empty signature")
	}
	if !Verify(secret, method, path, sig, ts, body) {
		t.Fatal("verify failed for matching sig")
	}
	if Verify(secret, method, path, sig, ts+1, body) {
		t.Fatal("verify should fail for different ts")
	}
	if Verify("wrong", method, path, sig, ts, body) {
		t.Fatal("verify should fail for wrong secret")
	}
}

func TestIsTimestampValid(t *testing.T) {
	now := time.Now().Unix()
	if !IsTimestampValid(now) {
		t.Fatal("now should be valid")
	}
	if !IsTimestampValid(now - 30) {
		t.Fatal("30s ago should be valid")
	}
	if IsTimestampValid(now - 120) {
		t.Fatal("120s ago should be invalid")
	}
}

func TestParseTimestamp(t *testing.T) {
	now := time.Now().Unix()
	s := strconv.FormatInt(now, 10)
	parsed, err := ParseTimestamp(s)
	if err != nil {
		t.Fatal(err)
	}
	if parsed != now {
		t.Fatalf("want %d got %d", now, parsed)
	}
}
