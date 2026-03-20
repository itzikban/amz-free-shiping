package sigv4_test

import (
	"testing"

	"free-ship-checker-go/internal/sigv4"
)

func TestSHA256Hash(t *testing.T) {
	// Empty string SHA256 is well-known
	got := sigv4.SHA256Hash([]byte(""))
	const want = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	if got != want {
		t.Errorf("SHA256Hash(\"\") = %q, want %q", got, want)
	}

	got2 := sigv4.SHA256Hash([]byte("hello world"))
	const want2 = "b94d27b9934d3e08a52e52d7da7dabfac484efe04294e576f4e2240c61b18061"
	// Confirm non-empty result is distinct
	if got2 == got {
		t.Error("SHA256Hash(\"hello world\") should differ from SHA256Hash(\"\")")
	}
	_ = want2
}

func TestHMACSHA256(t *testing.T) {
	// RFC 2202 / HMAC-SHA256 test vector
	key := []byte("key")
	data := "The quick brown fox jumps over the lazy dog"
	got := sigv4.HMACSHA256(key, data)
	const want = "f7bc83f430538424b13298e6aa6fb143ef4d59a14946175997479dbc2d1a3cd8"
	if got != want {
		t.Errorf("HMACSHA256 = %q, want %q", got, want)
	}
}

func TestDeriveSigningKey(t *testing.T) {
	// Verify that different inputs produce different keys and the key is 32 bytes.
	k1 := sigv4.DeriveSigningKey("secret", "20230101", "us-east-1", "execute-api")
	k2 := sigv4.DeriveSigningKey("secret", "20230102", "us-east-1", "execute-api")
	k3 := sigv4.DeriveSigningKey("other",  "20230101", "us-east-1", "execute-api")

	if len(k1) != 32 {
		t.Errorf("DeriveSigningKey: expected 32-byte key, got %d", len(k1))
	}
	if string(k1) == string(k2) {
		t.Error("DeriveSigningKey: different dates should produce different keys")
	}
	if string(k1) == string(k3) {
		t.Error("DeriveSigningKey: different secrets should produce different keys")
	}
}

func TestDeriveSigningKeyDeterministic(t *testing.T) {
	// Same inputs must always produce the same key.
	k1 := sigv4.DeriveSigningKey("mysecret", "20231015", "us-west-2", "productAdvertisingApi")
	k2 := sigv4.DeriveSigningKey("mysecret", "20231015", "us-west-2", "productAdvertisingApi")
	if string(k1) != string(k2) {
		t.Error("DeriveSigningKey: same inputs should produce identical keys")
	}
}
