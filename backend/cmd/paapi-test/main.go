package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"free-ship-checker-go/internal/sigv4"
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

	fmt.Println("Credentials loaded from environment (redacted)")
	fmt.Println()

	// PA-API endpoint
	host := "webservices.amazon.com"
	region := "us-east-1"
	service := "ProductAdvertisingAPI"
	endpoint := "https://webservices.amazon.com/paapi5/searchitems"

	// Build request body
	searchBody := map[string]any{
		"Keywords":    "wireless headphones",
		"ItemCount":   3,
		"PartnerTag":  associateTag,
		"PartnerType": "Associates",
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
	req.Header.Set("X-Amz-Target", "ProductAdvertisingAPI5.SearchItems")
	req.Header.Set("Host", host)

	// Sign request with AWS Signature v4
	now := time.Now().UTC()
	amzDate := now.Format("20060102T150405Z")
	datestamp := now.Format("20060102")

	// Create canonical request for signing
	payloadHash := sigv4.SHA256Hash(bodyBytes)

	// Headers to sign (sorted)
	signedHeaders := []string{"content-type", "host", "x-amz-date", "x-amz-target"}
	sort.Strings(signedHeaders)
	signedHeadersStr := strings.Join(signedHeaders, ";")

	canonicalHeaders := fmt.Sprintf(
		"content-type:%s\nhost:%s\nx-amz-date:%s\nx-amz-target:%s\n",
		req.Header.Get("Content-Type"),
		host,
		amzDate,
		req.Header.Get("X-Amz-Target"),
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
	canonicalHash := sigv4.SHA256Hash([]byte(canonicalRequest))
	stringToSign := fmt.Sprintf("AWS4-HMAC-SHA256\n%s\n%s\n%s", amzDate, credentialScope, canonicalHash)

	fmt.Printf("String to Sign:\n%s\n\n", stringToSign)

	// Calculate signature
	signingKey := sigv4.DeriveSigningKey(secretKey, datestamp, region, service)
	signature := sigv4.HMACSHA256(signingKey, stringToSign)

	fmt.Printf("Signature: %s\n\n", signature)

	// Add Authorization header
	authHeader := fmt.Sprintf(
		"AWS4-HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		accessKey, credentialScope, signedHeadersStr, signature,
	)
	req.Header.Set("Authorization", authHeader)
	req.Header.Set("X-Amz-Date", amzDate)

	fmt.Println("Authorization Header: <redacted>")
	fmt.Println()

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

	var result any
	if err := json.Unmarshal(respBody, &result); err != nil {
		fmt.Printf("Raw response:\n%s\n", string(respBody))
	} else {
		prettyJSON, _ := json.MarshalIndent(result, "", "  ")
		fmt.Printf("Response:\n%s\n", string(prettyJSON))
	}
}
