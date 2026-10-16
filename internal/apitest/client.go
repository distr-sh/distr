package apitest

import (
	"bytes"
	"cmp"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"strconv"
	"testing"
	"time"

	"github.com/onsi/gomega"
)

// Client sends requests to the API or the registry of a [Harness] with the credentials of one caller.
type Client struct {
	h             *Harness
	baseURL       string
	authorization string
	headers       map[string]string
}

// Anonymous is a client without credentials.
func (h *Harness) Anonymous() *Client {
	return &Client{h: h, baseURL: h.server.URL}
}

// WithBearer is a client sending token as a Bearer token, which is how users and controllers
// authenticate.
func (h *Harness) WithBearer(token string) *Client {
	return &Client{h: h, baseURL: h.server.URL, authorization: "Bearer " + token}
}

// WithAccessToken is a client authenticating with the personal access token key.
func (h *Harness) WithAccessToken(key string) *Client {
	return &Client{h: h, baseURL: h.server.URL, authorization: "AccessToken " + key}
}

func (h *Harness) WithBasicAuth(username, password string) *Client {
	return &Client{
		h:             h,
		baseURL:       h.server.URL,
		authorization: "Basic " + base64.StdEncoding.EncodeToString([]byte(username+":"+password)),
	}
}

// WithHeader is a copy of c that also sends the header key.
func (c *Client) WithHeader(key, value string) *Client {
	headers := maps.Clone(c.headers)
	if headers == nil {
		headers = map[string]string{}
	}
	headers[key] = value
	return &Client{h: c.h, baseURL: c.baseURL, authorization: c.authorization, headers: headers}
}

// Multipart is a multipart/form-data body. Fields are encoded as JSON, Values are written as they
// are and Files are uploaded as YAML documents, unless FileContentType is set.
type Multipart struct {
	Fields          map[string]any
	Values          map[string]string
	Files           map[string]string
	FileContentType string
}

func (m Multipart) encode() ([]byte, string, error) {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	for name, value := range m.Fields {
		if data, err := json.Marshal(value); err != nil {
			return nil, "", err
		} else if err := writer.WriteField(name, string(data)); err != nil {
			return nil, "", err
		}
	}
	for name, value := range m.Values {
		if err := writer.WriteField(name, value); err != nil {
			return nil, "", err
		}
	}
	contentType := cmp.Or(m.FileContentType, "application/yaml")
	for name, content := range m.Files {
		header := textproto.MIMEHeader{}
		header.Set("Content-Disposition", fmt.Sprintf(`form-data; name=%q; filename="%v.yaml"`, name, name))
		header.Set("Content-Type", contentType)
		if part, err := writer.CreatePart(header); err != nil {
			return nil, "", err
		} else if _, err := io.WriteString(part, content); err != nil {
			return nil, "", err
		}
	}
	if err := writer.Close(); err != nil {
		return nil, "", err
	}
	return buf.Bytes(), writer.FormDataContentType(), nil
}

// maxRateLimitWait is the longest Retry-After [Client.Do] waits for. A test easily exceeds the
// per-second limit of a user, but one hitting a longer limit has to fail rather than stall.
const maxRateLimitWait = 2 * time.Second

// Do sends body as a form if it is a [Multipart] and encoded as JSON otherwise, unless it is nil.
// A response rejected by a per-second rate limit is retried once the limit allows it.
func (c *Client) Do(method, path string, body any) *Response {
	t := c.h.t
	t.Helper()
	var data []byte
	var contentType string
	if body != nil {
		var err error
		if form, ok := body.(Multipart); ok {
			data, contentType, err = form.encode()
		} else {
			data, err = json.Marshal(body)
			contentType = "application/json"
		}
		if err != nil {
			t.Fatalf("failed to encode request body of %v %v: %v", method, path, err)
		}
	}
	for {
		resp := c.send(method, path, data, contentType)
		if resp.StatusCode != http.StatusTooManyRequests {
			return resp
		}
		wait, err := strconv.Atoi(resp.Header.Get("Retry-After"))
		if err != nil || time.Duration(wait)*time.Second > maxRateLimitWait {
			return resp
		}
		time.Sleep(time.Duration(max(wait, 1)) * time.Second)
	}
}

func (c *Client) send(method, path string, data []byte, contentType string) *Response {
	t := c.h.t
	t.Helper()
	var reader io.Reader
	if data != nil {
		reader = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(t.Context(), method, c.baseURL+path, reader)
	if err != nil {
		t.Fatalf("failed to create request %v %v: %v", method, path, err)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if c.authorization != "" {
		req.Header.Set("Authorization", c.authorization)
	}
	for key, value := range c.headers {
		req.Header.Set(key, value)
	}
	resp, err := c.h.server.Client().Do(req)
	if err != nil {
		t.Fatalf("%v %v failed: %v", method, path, err)
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response body of %v %v: %v", method, path, err)
	}
	return &Response{
		t:          t,
		request:    method + " " + path,
		StatusCode: resp.StatusCode,
		Header:     resp.Header,
		Body:       respBody,
	}
}

func (c *Client) Get(path string) *Response {
	c.h.t.Helper()
	return c.Do(http.MethodGet, path, nil)
}

func (c *Client) Post(path string, body any) *Response {
	c.h.t.Helper()
	return c.Do(http.MethodPost, path, body)
}

func (c *Client) Put(path string, body any) *Response {
	c.h.t.Helper()
	return c.Do(http.MethodPut, path, body)
}

func (c *Client) Patch(path string, body any) *Response {
	c.h.t.Helper()
	return c.Do(http.MethodPatch, path, body)
}

func (c *Client) Delete(path string) *Response {
	c.h.t.Helper()
	return c.Do(http.MethodDelete, path, nil)
}

type Response struct {
	t          testing.TB
	request    string
	StatusCode int
	Header     http.Header
	Body       []byte
}

// ExpectStatus fails the test unless the response has status.
func (r *Response) ExpectStatus(status int) *Response {
	r.t.Helper()
	gomega.NewWithT(r.t).Expect(r.StatusCode).To(gomega.Equal(status), "%v answered: %s", r.request, r.Body)
	return r
}

// Expect fails the test unless the response has status and returns its body decoded as T.
func Expect[T any](r *Response, status int) T {
	r.t.Helper()
	r.ExpectStatus(status)
	var body T
	gomega.NewWithT(r.t).Expect(json.Unmarshal(r.Body, &body)).To(gomega.Succeed(), "%v answered: %s", r.request, r.Body)
	return body
}
