package main

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func expect(t *testing.T, a interface{}, b interface{}) {
	if a != b {
		t.Errorf("Expected: %v (type %v)  Got: %v (type %v)", a, reflect.TypeOf(a), b, reflect.TypeOf(b))
	}
}

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

func TestComputeGridSquare(t *testing.T) {
	// W1AW location in CT
	result := ComputeGridSquare(41.7148, -72.7272)
	expect(t, "FN31pr", result)
	// Christ the Redeemer in Rio
	result = ComputeGridSquare(-22.951916, -43.2104872)
	expect(t, "GG87jb", result)
	// The Eiffel Tower
	result = ComputeGridSquare(55.7539303, 37.620795)
	expect(t, "KO85ts", result)
	// The Shire (New Zealand)
	result = ComputeGridSquare(-37.8720905, 175.6829096)
	expect(t, "RF72ud", result)
}
