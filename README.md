# BodyREST

A Go library for streamlined HTTP request handling with automatic path parameter parsing, request body binding, and field validation.

## Key Features

- Automatic JSON request body parsing into structs
- Multipart/form-data support
- Path parameter extraction (`{id}`, `{slug}` etc.) with type conversion
- Required field validation (for fields with JSON tags without `omitempty`)
- Customizable error handling
- Seamless integration with chi router

## Installation

```bash
go get github.com/ixalender/bodyrest
```

## Basic Usage

### Simple Example

```go
package main

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/ixalender/bodyrest"
)

type User struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

func main() {
	r := chi.NewRouter()

	r.Post("/users", bodyrest.HandleTo(createUser))

	http.ListenAndServe(":8080", r)
}

func createUser(u User) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Process user
		w.WriteHeader(http.StatusCreated)
	}
}
```

### Path Parameters Example

```go
func getUser(id int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Use ID from /users/{id} path
	}
}

r.Get("/users/{id}", bodyrest.HandleTo(getUser))
```

### Custom Error Handling

```go
func init() {
	bodyrest.SetRestErrorHandler(func(w http.ResponseWriter, r *http.Request, status int) {
		w.WriteHeader(status)
		w.Write([]byte(`{"error": "invalid request"}`))
	})
}
```

## How It Works

1. Analyzes handler function parameter types
2. For struct types:
   - Parses request body (JSON or multipart/form-data)
   - Validates required fields (without omitempty)
3. For primitive types (int, string, bool, float64):
   - Extracts values from path parameters
   - Performs type conversion
4. Handler must return exactly one value - http.HandlerFunc

## Requirements & Limitations

- Requires chi router for path parameter functionality
- Only POST/PUT/PATCH requests can have body payloads
- Handler must return http.HandlerFunc
- Supported path parameter types: int, string, bool, float64

## License

MIT - See LICENSE file for details.
