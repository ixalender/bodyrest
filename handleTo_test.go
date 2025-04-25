package bodyrest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

type testHandler struct{}

type testHandlerRequest struct {
	Message    string  `json:"message"`
	MessagePtr *string `json:"messagePtr"`
	Code       int     `json:"code"`
	CodePtr    *int    `json:"codePtr"`
}

func (h *testHandler) testPost(req testHandlerRequest) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}
}

func (h *testHandler) testPostWithParamsAndBody(id int, req testHandlerRequest) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}
}

func (h *testHandler) wrongTestPostWithZeroParams() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}
}

func (h *testHandler) wrongTestPost(req testHandlerRequest) string {
	return "wrong"
}

func (h *testHandler) wrongTestPostWithNoReturn(req testHandlerRequest) {
}
func (h *testHandler) wrongTestPostWithNoReturnAndNoBody() {
}

type ErrorAnswer struct {
	Message string `json:"message"`
}

var (
	ErrHttpBadRequestText    = "Error while parsing request. Please check your request and try again."
	ErrHttpInternalErrorText = "Something went wrong. Please try again later."
)

func init() {
	SetRestErrorHandler(func(w http.ResponseWriter, r *http.Request, status int) {
		w.WriteHeader(status)
		switch status {
		case http.StatusBadRequest:
			json.NewEncoder(w).Encode(ErrorAnswer{Message: ErrHttpBadRequestText})
		case http.StatusInternalServerError:
			json.NewEncoder(w).Encode(ErrorAnswer{Message: ErrHttpInternalErrorText})
		default:
			json.NewEncoder(w).Encode(ErrorAnswer{Message: ErrHttpInternalErrorText})
		}
	})
}

func TestHandleTo(t *testing.T) {
	testHandler := &testHandler{}

	testCases := []struct {
		name           string
		jsonPayload    string
		expectedStatus int
		expectedBody   string
		handler        interface{}
		pattern        string
		path           string
	}{
		{
			name:           "Valid JSON payload",
			jsonPayload:    `{"message":"Hello", "code": 200, "messagePtr": "Hello", "codePtr": 200}`,
			expectedStatus: http.StatusOK,
			expectedBody:   ``,
			handler:        testHandler.testPost,
			pattern:        "/test",
			path:           "/test",
		},
		{
			name:           "With no messagePtr",
			jsonPayload:    `{"message":"Hello", "code": 200, "codePtr": 200}`,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   fmt.Sprintf(`{"message":"%s"}`, ErrHttpBadRequestText),
			handler:        testHandler.testPost,
			pattern:        "/test",
			path:           "/test",
		},
		{
			name:           "With no Ptrs",
			jsonPayload:    `{"message":"Hello", "code": 200}`,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   fmt.Sprintf(`{"message":"%s"}`, ErrHttpBadRequestText),
			handler:        testHandler.testPost,
			pattern:        "/test",
			path:           "/test",
		},
		{
			name:           "With wrong messagePtr",
			jsonPayload:    `{"message":"Hello", "code": 200, "messagePtr": 11, "codePtr": 200}`,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   fmt.Sprintf(`{"message":"%s"}`, ErrHttpBadRequestText),
			handler:        testHandler.testPost,
			pattern:        "/test",
			path:           "/test",
		},
		{
			name:           "With wrong codePtr",
			jsonPayload:    `{"message":"Hello", "code": 200, "messagePtr": "hello", "codePtr": "200"}`,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   fmt.Sprintf(`{"message":"%s"}`, ErrHttpBadRequestText),
			handler:        testHandler.testPost,
			pattern:        "/test",
			path:           "/test",
		},
		{
			name:           "Empty JSON payload",
			jsonPayload:    `{}`,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   fmt.Sprintf(`{"message":"%s"}`, ErrHttpBadRequestText),
			handler:        testHandler.testPost,
			pattern:        "/test",
			path:           "/test",
		},
		{
			name:           "Empty payload",
			jsonPayload:    ``,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   fmt.Sprintf(`{"message":"%s"}`, ErrHttpBadRequestText),
			handler:        testHandler.testPost,
			pattern:        "/test",
			path:           "/test",
		},
		{
			name:           "Valid JSON payload to handler with param and body",
			jsonPayload:    `{"message":"Hello", "code": 200, "messagePtr": "Hello", "codePtr": 200}`,
			expectedStatus: http.StatusOK,
			expectedBody:   ``,
			handler:        testHandler.testPostWithParamsAndBody,
			pattern:        "/test/{id}",
			path:           "/test/1",
		},
		{
			name:           "Valid JSON payload to handler with invalid param and body",
			jsonPayload:    `{"message":"Hello", "code": 200, "messagePtr": "Hello", "codePtr": 200}`,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   fmt.Sprintf(`{"message":"%s"}`, ErrHttpBadRequestText),
			handler:        testHandler.testPostWithParamsAndBody,
			pattern:        "/test/{id}",
			path:           "/test/test",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest("POST", tc.path, bytes.NewBufferString(tc.jsonPayload))
			if err != nil {
				t.Fatal(err)
			}

			r := chi.NewRouter()
			r.Post(tc.pattern, HandleTo(tc.handler))
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != tc.expectedStatus {
				t.Errorf("Expected status code %d, got %d", tc.expectedStatus, w.Code)
			}

			if strings.TrimSpace(w.Body.String()) != strings.TrimSpace(tc.expectedBody) {
				t.Errorf("Expected body %s, got %s", tc.expectedBody, w.Body.String())
			}
		})
	}
}

func TestWrongHandleTo(t *testing.T) {
	testHandler := &testHandler{}

	testCases := []struct {
		name           string
		jsonPayload    string
		expectedStatus int
		expectedBody   string
		pattern        string
		path           string
	}{
		{
			name:           "Invalid handler but with valid JSON payload",
			jsonPayload:    `{"message":"Hello", "code": 200, "messagePtr": "Hello", "codePtr": 200}`,
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   fmt.Sprintf(`{"message":"%s"}`, ErrHttpInternalErrorText),
			pattern:        "/test",
			path:           "/test",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest("POST", tc.path, bytes.NewBufferString(tc.jsonPayload))
			if err != nil {
				t.Fatal(err)
			}

			w := httptest.NewRecorder()
			r := chi.NewRouter()
			r.Post(tc.pattern, HandleTo(testHandler.wrongTestPost))
			r.ServeHTTP(w, req)

			if w.Code != tc.expectedStatus {
				t.Errorf("Expected status code %d, got %d", tc.expectedStatus, w.Code)
			}

			if strings.TrimSpace(w.Body.String()) != strings.TrimSpace(tc.expectedBody) {
				t.Errorf("Expected body %s, got %s", tc.expectedBody, w.Body.String())
			}
		})
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest("POST", tc.path, bytes.NewBufferString(tc.jsonPayload))
			if err != nil {
				t.Fatal(err)
			}

			w := httptest.NewRecorder()
			r := chi.NewRouter()
			r.Post(tc.pattern, HandleTo(testHandler.wrongTestPostWithNoReturn))
			r.ServeHTTP(w, req)

			if w.Code != tc.expectedStatus {
				t.Errorf("Expected status code %d, got %d", tc.expectedStatus, w.Code)
			}

			if strings.TrimSpace(w.Body.String()) != strings.TrimSpace(tc.expectedBody) {
				t.Errorf("Expected body %s, got %s", tc.expectedBody, w.Body.String())
			}
		})
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest("POST", tc.path, bytes.NewBufferString(tc.jsonPayload))
			if err != nil {
				t.Fatal(err)
			}

			w := httptest.NewRecorder()
			r := chi.NewRouter()
			r.Post(tc.pattern, HandleTo(testHandler.wrongTestPostWithNoReturnAndNoBody))
			r.ServeHTTP(w, req)

			if w.Code != tc.expectedStatus {
				t.Errorf("Expected status code %d, got %d", tc.expectedStatus, w.Code)
			}

			if strings.TrimSpace(w.Body.String()) != strings.TrimSpace(tc.expectedBody) {
				t.Errorf("Expected body %s, got %s", tc.expectedBody, w.Body.String())
			}
		})
	}
}
