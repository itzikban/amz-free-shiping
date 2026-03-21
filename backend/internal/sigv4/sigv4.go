package sigv4

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

// SHA256Hash returns hex-encoded SHA256 hash.
func SHA256Hash(data []byte) string {
	h := sha256.New()
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}

// HMACSHA256 returns hex-encoded HMAC-SHA256.
func HMACSHA256(key []byte, data string) string {
	h := hmac.New(sha256.New, key)
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

// DeriveSigningKey derives AWS Signature V4 signing key.
func DeriveSigningKey(secretKey, dateStamp, region, service string) []byte {
	kDate := hmac.New(sha256.New, []byte("AWS4"+secretKey))
	kDate.Write([]byte(dateStamp))

	kRegion := hmac.New(sha256.New, kDate.Sum(nil))
	kRegion.Write([]byte(region))

	kService := hmac.New(sha256.New, kRegion.Sum(nil))
	kService.Write([]byte(service))

	kSigning := hmac.New(sha256.New, kService.Sum(nil))
	kSigning.Write([]byte("aws4_request"))

	return kSigning.Sum(nil)
}
