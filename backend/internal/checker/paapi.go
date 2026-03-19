package checker

import (
	"bytes"
	"context"
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

// PAAClient holds PA-API v5 credentials and client
type PAAClient struct {
	AccessKeyID string
	SecretKey   string
	AssociateTag string
	Region      string
	HTTPClient  *http.Client
}

// NewPAAClient creates a PA-API client from environment variables
func NewPAAClient() *PAAClient {
	return &PAAClient{
		AccessKeyID:  os.Getenv("AWS_ACCESS_KEY_ID"),
		SecretKey:    os.Getenv("AWS_SECRET_ACCESS_KEY"),
		AssociateTag: os.Getenv("AWS_ASSOCIATE_TAG"),
		Region:       "us-east-1",
		HTTPClient:   &http.Client{Timeout: 10 * time.Second},
	}
}

// SearchItemsResponse models PA-API v5 response
type SearchItemsResponse struct {
	Errors []struct {
		Code    string `json:"Code"`
		Message string `json:"Message"`
	} `json:"Errors"`
	SearchResult struct {
		Items []struct {
			ASIN string `json:"ASIN"`
			ItemInfo struct {
				Title struct {
					DisplayValue string `json:"DisplayValue"`
				} `json:"Title"`
			} `json:"ItemInfo"`
			Images struct {
				Primary struct {
					Medium struct {
						URL string `json:"URL"`
					} `json:"Medium"`
				} `json:"Primary"`
			} `json:"Images"`
			Offers struct {
				Listings []struct {
					Price struct {
						Amount   float64 `json:"Amount"`
						Currency string  `json:"Currency"`
					} `json:"Price"`
					DeliveryInfo struct {
						IsFreeShippingEligible bool `json:"IsFreeShippingEligible"`
					} `json:"DeliveryInfo"`
				} `json:"Listings"`
			} `json:"Offers"`
		} `json:"Items"`
	} `json:"SearchResult"`
}

// SearchItems calls PA-API v5 SearchItems operation
func (c *PAAClient) SearchItems(ctx context.Context, keywords string, itemCount int) ([]Alternative, error) {
	if c.AccessKeyID == "" || c.SecretKey == "" || c.AssociateTag == "" {
		return nil, fmt.Errorf("missing AWS credentials")
	}

	endpoint := "https://webservices.amazon.com/paapi5/searchitems"
	host := "webservices.amazon.com"
	service := "ProductAdvertisingAPI"

	// Build request body
	reqBody := map[string]any{
		"Keywords":    keywords,
		"ItemCount":   itemCount,
		"PartnerTag":  c.AssociateTag,
		"PartnerType": "Associates",
		"Resources": []string{
			"Images.Primary.Medium",
			"ItemInfo.Title",
			"Offers.Listings.Price",
			"Offers.Listings.DeliveryInfo.IsFreeShippingEligible",
		},
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Content-Encoding", "amz-1.0")
	req.Header.Set("X-Amz-Target", "com.amazon.paapi5.v1.ProductAdvertisingAPIv1.SearchItems")
	req.Header.Set("Host", host)

	// Sign request with AWS Signature v4
	c.signRequest(req, bodyBytes, host, service)

	// Execute request
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("pa-api error: %d %s", resp.StatusCode, string(respBody))
	}

	// Parse response
	var result SearchItemsResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}
	if len(result.Errors) > 0 {
		return nil, fmt.Errorf("pa-api response error: %s: %s", result.Errors[0].Code, result.Errors[0].Message)
	}

	// Extract alternatives with free shipping only
	var alternatives []Alternative
	for _, item := range result.SearchResult.Items {
		if len(item.Offers.Listings) == 0 {
			continue
		}

		selectedIdx := -1
		for i, listing := range item.Offers.Listings {
			if listing.DeliveryInfo.IsFreeShippingEligible {
				selectedIdx = i
				break
			}
		}
		if selectedIdx == -1 {
			continue
		}
		listing := item.Offers.Listings[selectedIdx]

		alt := Alternative{
			ASIN:         item.ASIN,
			Title:        item.ItemInfo.Title.DisplayValue,
			URL:          fmt.Sprintf("https://amazon.com/dp/%s", item.ASIN),
			ImageURL:     item.Images.Primary.Medium.URL,
			PriceUSD:     listing.Price.Amount,
			FreeShipping: true,
		}
		alternatives = append(alternatives, alt)
	}

	return alternatives, nil
}

// signRequest signs HTTP request with AWS Signature Version 4
func (c *PAAClient) signRequest(req *http.Request, bodyBytes []byte, host, service string) {
	now := time.Now().UTC()
	amzDate := now.Format("20060102T150405Z")
	datestamp := now.Format("20060102")

	payloadHash := sha256Hash(bodyBytes)

	// Canonical request
	signedHeaders := []string{"content-encoding", "content-type", "host", "x-amz-date", "x-amz-target"}
	sort.Strings(signedHeaders)
	signedHeadersStr := strings.Join(signedHeaders, ";")

	canonicalHeaders := fmt.Sprintf(
		"content-encoding:%s\ncontent-type:%s\nhost:%s\nx-amz-date:%s\nx-amz-target:%s\n",
		req.Header.Get("Content-Encoding"),
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

	// String to sign
	credentialScope := fmt.Sprintf("%s/%s/%s/aws4_request", datestamp, c.Region, service)
	canonicalHash := sha256Hash([]byte(canonicalRequest))
	stringToSign := fmt.Sprintf("AWS4-HMAC-SHA256\n%s\n%s\n%s", amzDate, credentialScope, canonicalHash)

	// Derive signing key
	signingKey := deriveSigningKey(c.SecretKey, datestamp, c.Region, service)
	signature := hmacSha256(signingKey, stringToSign)

	// Build Authorization header
	authHeader := fmt.Sprintf(
		"AWS4-HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		c.AccessKeyID, credentialScope, signedHeadersStr, signature,
	)

	req.Header.Set("Authorization", authHeader)
	req.Header.Set("X-Amz-Date", amzDate)
}

// Helper functions for AWS Signature v4

func sha256Hash(data []byte) string {
	h := sha256.New()
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}

func hmacSha256(key []byte, data string) string {
	h := hmac.New(sha256.New, key)
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

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
