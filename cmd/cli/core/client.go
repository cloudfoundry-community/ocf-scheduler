package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/ess/hype"
)

type Driver struct {
	raw         *hype.Driver
	baseURL     string
	token       string
	accept      *hype.Header
	contentType *hype.Header
	auth        *hype.Header
}

var Client *Driver

// fetchClient waits as long as gorouter does by default, because an orphan
// list asks Cloud Controller about every app before it answers.
var fetchClient = &http.Client{Timeout: 15 * time.Minute}

// maxMessage bounds the text shown from a body that isn't the scheduler's
// own error, such as a gateway's HTML page.
const maxMessage = 512

func NewDriver(baseURL string, token string) (*Driver, error) {
	raw, err := hype.New(baseURL)
	if err != nil {
		return nil, err
	}

	d := &Driver{
		raw,
		baseURL,
		token,
		hype.NewHeader("Accept", "application/json"),
		hype.NewHeader("Content-Type", "application/json"),
		hype.NewHeader("Authorization", token),
	}

	return d, nil
}

func (driver *Driver) Token() string {
	return driver.token
}

func (driver *Driver) Delete(path string, params hype.Params) hype.Response {
	return driver.
		raw.
		Delete(path, params).
		WithHeaderSet(driver.accept, driver.contentType, driver.auth).
		Response()
}

func (driver *Driver) Get(path string, params hype.Params) hype.Response {
	return driver.
		raw.
		Get(path, params).
		WithHeaderSet(driver.accept, driver.contentType, driver.auth).
		Response()
}

func (driver *Driver) Post(path string, params hype.Params, data []byte) hype.Response {
	return driver.
		raw.
		Post(path, params, data).
		WithHeaderSet(driver.accept, driver.contentType, driver.auth).
		Response()
}

// Result is a response whose body is kept whatever the status, which the
// hype response does not do for a status outside 2xx.
type Result struct {
	Status int
	Body   []byte
}

func (result *Result) Okay() bool {
	return result.Status >= 200 && result.Status < 300
}

// Error describes a failed response, with the message that the scheduler
// sent when there is one.
func (result *Result) Error() error {
	message := Message(result.Body)
	if message == "" {
		return fmt.Errorf("response status: %d", result.Status)
	}

	return fmt.Errorf("response status: %d: %s", result.Status, message)
}

// Message extracts the explanation from an error body. The scheduler sends
// either a JSON string or an object with a list of errors, and a gateway may
// send plain text.
func Message(body []byte) string {
	var text string
	if json.Unmarshal(body, &text) == nil {
		return strings.TrimSpace(text)
	}

	var findings struct {
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if json.Unmarshal(body, &findings) == nil && len(findings.Errors) > 0 {
		messages := make([]string, 0, len(findings.Errors))
		for _, finding := range findings.Errors {
			messages = append(messages, finding.Message)
		}

		return strings.Join(messages, "; ")
	}

	message := strings.TrimSpace(string(body))
	if len(message) > maxMessage {
		message = message[:maxMessage] + "..."
	}

	return message
}

// Fetch makes a request and keeps the body of the response, even when the
// status is an error.
func (driver *Driver) Fetch(method, path string, body []byte) (*Result, error) {
	url := strings.TrimRight(driver.baseURL, "/") + "/" + strings.TrimLeft(path, "/")

	request, err := http.NewRequest(method, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	request.Header.Set("Accept", "application/json")
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", driver.token)

	response, err := fetchClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	data, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	return &Result{Status: response.StatusCode, Body: data}, nil
}
