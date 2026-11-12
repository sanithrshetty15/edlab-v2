package twilio

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const verifyBaseURL = "https://verify.twilio.com/v2"

// ErrNotConfigured is returned when Twilio Verify credentials are missing.
var ErrNotConfigured = errors.New("Twilio Verify credentials are not configured")

// Client talks to the Twilio Verify API.
type Client struct {
	AccountSID string
	AuthToken  string
	ServiceSID string
	HTTP       *http.Client
}

// NewFromEnv builds a client from Twilio environment variables.
func NewFromEnv() *Client {
	return &Client{
		AccountSID: strings.TrimSpace(os.Getenv("TWILIO_ACCOUNT_SID")),
		AuthToken:  strings.TrimSpace(os.Getenv("TWILIO_AUTH_TOKEN")),
		ServiceSID: strings.TrimSpace(os.Getenv("TWILIO_VERIFY_SERVICE_SID")),
		HTTP:       &http.Client{Timeout: 15 * time.Second},
	}
}

func (c *Client) Configured() bool {
	return c != nil && c.AccountSID != "" && c.AuthToken != "" && c.ServiceSID != ""
}

// SendVerification requests an SMS verification code.
func (c *Client) SendVerification(phoneNumber string) error {
	if !c.Configured() {
		return ErrNotConfigured
	}
	_, err := c.post("/Services/"+url.PathEscape(c.ServiceSID)+"/Verifications", url.Values{
		"To":      {phoneNumber},
		"Channel": {"sms"},
	})
	return err
}

// CheckVerification validates a code previously sent by Twilio.
func (c *Client) CheckVerification(phoneNumber, code string) (bool, error) {
	if !c.Configured() {
		return false, ErrNotConfigured
	}
	result, err := c.post("/Services/"+url.PathEscape(c.ServiceSID)+"/VerificationCheck", url.Values{
		"To":   {phoneNumber},
		"Code": {code},
	})
	if err != nil {
		return false, err
	}
	return strings.EqualFold(strings.TrimSpace(fmt.Sprint(result["status"])), "approved"), nil
}

func (c *Client) post(path string, form url.Values) (map[string]any, error) {
	request, err := http.NewRequest(http.MethodPost, verifyBaseURL+path, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	request.SetBasicAuth(c.AccountSID, c.AuthToken)
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	response, err := c.HTTP.Do(request)
	if err != nil {
		return nil, fmt.Errorf("Twilio request failed: %w", err)
	}
	defer response.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(response.Body, 8192))
	var result map[string]any
	if len(body) > 0 {
		if err := json.Unmarshal(body, &result); err != nil {
			return nil, errors.New("Twilio returned an invalid response")
		}
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		message := ""
		if result != nil {
			message = strings.TrimSpace(fmt.Sprint(result["message"]))
		}
		if message == "" {
			message = response.Status
		}
		return nil, fmt.Errorf("Twilio verification failed: %s", message)
	}
	return result, nil
}
