package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"
)

func main() {
	// Read credentials from env
	accessKey := os.Getenv("AWS_ACCESS_KEY_ID")
	secretKey := os.Getenv("AWS_SECRET_ACCESS_KEY")
	associateTag := os.Getenv("AWS_ASSOCIATE_TAG")

	if accessKey == "" || secretKey == "" || associateTag == "" {
		fmt.Println("Error: AWS credentials not set in environment")
		os.Exit(1)
	}

	fmt.Printf("Access Key: %s\n", accessKey[:10]+"...")
	fmt.Printf("Associate Tag: %s\n\n", associateTag)

	// PA-API endpoint
	host := "webservices.amazon.com"
	region := "us-east-1"
	service := "ProductAdvertisingAPI"
	endpoint := "https://webservices.amazon.com/paapi5/searchitems"

	// Build request body
	searchBody := map[string]interface{}{
		"Keywords":    "wireless headphones",
		"ItemCount":   3,
		"PartnerTag":  associateTag,
		"Resources": []string{
			"Images.Primary.Medium",
			"ItemInfo.Title",
			"Offers.Listings.Price",
			"Offers.Listings.DeliveryInfo.IsFreeShippingEligible",
		},
	}

	bodyBytes, err := json.Marshal(searchBody)
	if err != nil {
		fmt.Printf("Error marshaling request: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Request body:\n%s\n\n", string(bodyBytes))

	// Create HTTP request
	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		fmt.Printf("Error creating request: %v\n", err)
		os.Exit(1)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Host", host)

	// Sign request with AWS Signature v4
	now := time.Now().UTC()
	amzDate := now.Format("20060102T150405Z")
	datestamp := now.Format("20060102")

	// Create canonical request for signing
	payloadHash := sha256Hash(bodyBytes)

	// Headers to sign (sorted)
	signedHeaders := []string{"content-type", "host", "x-amz-date"}
	sort.Strings(signedHeaders)
	signedHeadersStr := strings.Join(signedHeaders, ";")

	canonicalHeaders := fmt.Sprintf(
		"content-type:%s\nhost:%s\nx-amz-date:%s\n",
		req.Header.Get("Content-Type"),
		host,
		amzDate,
	)

	canonicalRequest := fmt.Sprintf(
		"%s\n%s\n%s\n%s\n%s\n%s",
		req.Method,
		"/paapi5/searchitems",
		"",
		canonicalHeaders,
		signedHeadersStr,
		payloadHash,
	)

	fmt.Printf("Canonical Request:\n%s\n\n", canonicalRequest)

	// Create string to sign
	credentialScope := fmt.Sprintf("%s/%s/%s/aws4_request", datestamp, region, service)
	canonicalHash := sha256Hash([]byte(canonicalRequest))
	stringToSign := fmt.Sprintf("AWS4-HMAC-SHA256\n%s\n%s\n%s", amzDate, credentialScope, canonicalHash)

	fmt.Printf("String to Sign:\n%s\n\n", stringToSign)

	// Calculate signature
	signingKey := deriveSigningKey(secretKey, datestamp, region, service)
	signature := hmacSha256(signingKey, stringToSign)

	fmt.Printf("Signature: %s\n\n", signature)

	// Add Authorization header
	authHeader := fmt.Sprintf(
		"AWS4-HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		accessKey, credentialScope, signedHeadersStr, signature,
	)
	req.Header.Set("Authorization", authHeader)
	req.Header.Set("X-Amz-Date", amzDate)

	fmt.Printf("Authorization Header: %s\n\n", authHeader)

	// Make request
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error making request: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading response: %v\n", err)
		os.Exit(1)
	}

	// Pretty print response
	fmt.Printf("Status Code: %d\n\n", resp.StatusCode)

	if resp.StatusCode >= 400 {
		fmt.Printf("Error Response:\n%s\n", string(respBody))
		return
	}

	var result interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		fmt.Printf("Raw response:\n%s\n", string(respBody))
	} else {
		prettyJSON, _ := json.MarshalIndent(result, "", "  ")
		fmt.Printf("Response:\n%s\n", string(prettyJSON))
	}
}

// sha256Hash returns hex-encoded SHA256 hash
func sha256Hash(data []byte) string {
	h := sha256.New()
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}

// hmacSha256 returns hex-encoded HMAC-SHA256
func hmacSha256(key []byte, data string) string {
	h := hmac.New(sha256.New, key)
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

// deriveSigningKey derives the AWS Signature v4 signing key
func deriveSigningKey(secretKey, dateStamp, region, service string) []byte {
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
