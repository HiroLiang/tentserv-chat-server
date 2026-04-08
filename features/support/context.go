package support

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/HiroLiang/tentserv-chat-server/internal/interface/http/response"
	"github.com/cucumber/godog"
)

type APITestContext struct {
	Client       *http.Client
	BaseURL      string
	Response     *http.Response
	ResponseBody []byte
}

func NewAPITestContext(baseURL string) *APITestContext {
	return &APITestContext{
		BaseURL: baseURL,
		Client:  &http.Client{},
	}
}

func (a *APITestContext) Reset() {
	a.Response = nil
	a.ResponseBody = nil
}

func RegisterCommonSteps(ctx *godog.ScenarioContext, apiCtx *APITestContext) {
	ctx.Step(`^the response status should be (\d+)$`, apiCtx.theResponseStatusShouldBe)
	ctx.Step(`^the response error code should be "([^"]*)"$`, apiCtx.theResponseErrorCodeShouldBe)
}

func (a *APITestContext) DoJSONRequest(method, path string, payload map[string]string) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(method, a.BaseURL+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.Client.Do(req)
	if err != nil {
		return err
	}
	a.Response = resp
	a.ResponseBody, _ = io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	return nil
}

func (a *APITestContext) DoRequest(method, path string) error {
	req, err := http.NewRequest(method, a.BaseURL+path, nil)
	if err != nil {
		return err
	}

	resp, err := a.Client.Do(req)
	if err != nil {
		return err
	}
	a.Response = resp
	a.ResponseBody, _ = io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	return nil
}

func (a *APITestContext) theResponseStatusShouldBe(status int) error {
	start := time.Now()
	if a.Response == nil {
		return fmt.Errorf("no response captured")
	}

	fmt.Println("Given: an HTTP response was captured")
	fmt.Printf("Input: expected_status=%d actual_status=%d\n", status, a.Response.StatusCode)
	fmt.Println("Action: compare response status")
	fmt.Printf("Output: status_match=%t\n", a.Response.StatusCode == status)
	fmt.Println("Mutation: none")
	fmt.Printf("Duration: %s\n", time.Since(start))

	if a.Response.StatusCode != status {
		return fmt.Errorf("expected status %d, got %d body=%s", status, a.Response.StatusCode, string(a.ResponseBody))
	}
	return nil
}

func (a *APITestContext) theResponseErrorCodeShouldBe(code string) error {
	start := time.Now()
	fmt.Println("Given: response body should contain an error response")
	fmt.Printf("Input: expected_error_code=%s\n", code)
	fmt.Println("Action: decode and compare response error code")

	var errResp response.ErrorResponse
	if err := json.Unmarshal(a.ResponseBody, &errResp); err != nil {
		return fmt.Errorf("decode response error: %w; body=%s", err, string(a.ResponseBody))
	}

	fmt.Printf("Output: actual_error_code=%s match=%t\n", errResp.Code, errResp.Code == code)
	fmt.Println("Mutation: none")
	fmt.Printf("Duration: %s\n", time.Since(start))

	if errResp.Code != code {
		return fmt.Errorf("expected error code %s, got %s", code, errResp.Code)
	}
	return nil
}
