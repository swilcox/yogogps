package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGETHome(t *testing.T) {
	request, _ := http.NewRequest(http.MethodGet, "/", nil)
	response := httptest.NewRecorder()

	home(response, request)
	got := response.Body.String()
	expect := "<!DOCTYPE html>"
	if !strings.Contains(got, expect) {
		t.Errorf("response did not contain: %q", expect)
	}
}
