package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"

	"moetopia.local/aio/api/internal/httpserver"
)

func main() {
	response := httptest.NewRecorder()
	httpserver.New(httpserver.Dependencies{}).ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/openapi.json", nil))
	if response.Code != http.StatusOK {
		fmt.Fprintln(os.Stderr, "OpenAPI generation failed")
		os.Exit(1)
	}
	_, _ = os.Stdout.Write(response.Body.Bytes())
}
