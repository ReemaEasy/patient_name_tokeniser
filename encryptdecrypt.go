package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"log"
	"net/http"
)

type Client struct {
	httpClient  *DummyClient
	config      TokenizerService
	httpOptions map[string]string
}
type RxClient struct {
	httpClient *DummyClient
	config     RxService
}

type TokenizerService struct {
	Identifier string
	BaseURL    string
}
type RxService struct {
	BaseURL string
	Origin  Origin
}

type Token struct {
	Token string `json:"token"`
}

type DecryptRequest []Token

type DecryptResponse struct {
	Data []struct {
		Token    string `json:"token"`
		Content  string `json:"content"`  // Fix here
		Metadata any    `json:"metadata"` // Optional: if you want to access it
	} `json:"data"`
}

const (
	urlDecrypt = "%s/v1/decrypt"
	serviceUrl = "%s/api/v1/tenants/%s/customers/%s/prescriptions/%s"
)

func (c *Client) getHeaders() map[string]string {
	return map[string]string{
		"Content-Type": "application/json",
	}
}

func (tokens *DecryptRequest) prepareDecryptRequestBody(identifier string) map[string]interface{} {
	return map[string]interface{}{
		"identifier": identifier,
		"requestId":  uuid.New().String(),
		"data":       tokens,
	}
}

func (c *Client) Decrypt(ctx context.Context, tokens *DecryptRequest) (*DecryptResponse, error) {
	url := fmt.Sprintf(urlDecrypt, c.config.BaseURL)

	body := tokens.prepareDecryptRequestBody(c.config.Identifier)

	// Print request body
	if reqBytes, err := json.MarshalIndent(body, "", "  "); err == nil {
		log.Println("Decrypt request body:")
		log.Println(string(reqBytes))
	} else {
		log.Println("Failed to marshal Decrypt request body:", err)
	}

	resp, err := c.httpClient.NewPostRequest(ctx, url, c.getHeaders(), body)
	if err != nil {
		return nil, err
	}

	// Print raw response body
	log.Println("Decrypt API raw response body:")
	log.Println(string(resp.Body))

	var result DecryptResponse
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		return nil, errors.New("failed to parse Decrypt response")
	}

	return &result, nil
}
func (c *RxClient) UpdatePatient(ctx context.Context, rxId, tenantId, customerId string, body UpdateRequestBody) ([]byte, error) {
	url := fmt.Sprintf(serviceUrl, c.config.BaseURL, tenantId, customerId, rxId)

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("tenantId", tenantId)
	req.Header.Set("customerId", customerId)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error: %s", resp.Body)
	}

	var rxResponse struct {
		Patient struct {
			Name       string `json:"name"`
			HashedName string `json:"hashedName"`
		} `json:"patient"`
	}

	if err := json.Unmarshal(resp.Body, &rxResponse); err != nil {
		return nil, fmt.Errorf("failed to unmarshal patient info: %w", err)
	}

	log.Printf("Updated patient name: %s", rxResponse.Patient.Name)
	log.Printf("Updated patient hashedName: %s", rxResponse.Patient.HashedName)

	return resp.Body, nil
}
