package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
)

type DummyClient struct{}

func NewClient() *DummyClient {
	return &DummyClient{}
}

type DummyResponse struct {
	Status int
	Body   []byte
}
type HttpResponse struct {
	Body       []byte
	StatusCode int
}

func (c *DummyClient) NewPostRequest(ctx context.Context, url string, headers map[string]string, body interface{}) (*DummyResponse, error) {
	// Replace this logic with actual HTTP call if needed
	jsonBody, _ := json.Marshal(body)
	req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBytes, _ := ioutil.ReadAll(resp.Body)

	return &DummyResponse{
		Status: resp.StatusCode,
		Body:   respBytes,
	}, nil
}

func (c *DummyClient) Do(req *http.Request) (*HttpResponse, error) {
	client := http.DefaultClient
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return &HttpResponse{
		Body:       respBytes,
		StatusCode: resp.StatusCode,
	}, nil
}
