package testutil

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"

	"github.com/gin-gonic/gin"
)

// TestRequest represents a test HTTP request
type TestRequest struct {
	Method      string
	Path        string
	Headers     map[string]string
	Body        interface{}
	QueryParams map[string]string
}

// TestResponse represents a test HTTP response
type TestResponse struct {
	StatusCode int
	Body       []byte
	Headers    http.Header
}

// ExecuteRequest executes an HTTP request against a gin router and returns the response
func ExecuteRequest(router *gin.Engine, req *TestRequest) (*TestResponse, error) {
	var bodyReader io.Reader
	if req.Body != nil {
		bodyBytes, err := json.Marshal(req.Body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	request, err := http.NewRequest(req.Method, req.Path, bodyReader)
	if err != nil {
		return nil, err
	}

	// Add headers
	for k, v := range req.Headers {
		request.Header.Set(k, v)
	}

	// Add query params
	if len(req.QueryParams) > 0 {
		q := request.URL.Query()
		for k, v := range req.QueryParams {
			q.Add(k, v)
		}
		request.URL.RawQuery = q.Encode()
	}

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	return &TestResponse{
		StatusCode: recorder.Code,
		Body:       recorder.Body.Bytes(),
		Headers:    recorder.Header(),
	}, nil
}

// ExecuteRequestWithToken executes a request with an Authorization header
func ExecuteRequestWithToken(router *gin.Engine, method, path, token string, body interface{}) (*TestResponse, error) {
	req := &TestRequest{
		Method:  method,
		Path:    path,
		Body:    body,
		Headers: map[string]string{"Authorization": "Bearer " + token},
	}
	return ExecuteRequest(router, req)
}

// ExecuteRequestWithQuery executes a request with query parameters
func ExecuteRequestWithQuery(router *gin.Engine, method, path string, query map[string]string, body interface{}) (*TestResponse, error) {
	req := &TestRequest{
		Method:      method,
		Path:        path,
		QueryParams: query,
		Body:        body,
	}
	return ExecuteRequest(router, req)
}