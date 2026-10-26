package netinfo

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNormalizeMAC(t *testing.T) {
	valid := map[string]string{
		"00:1a:2b:3c:4d:5e": "001A2B3C4D5E",
		"00-1A-2B-3C-4D-5E": "001A2B3C4D5E",
		"001a.2b3c.4d5e":    "001A2B3C4D5E",
		"00:1a:2b":          "001A2B",
	}
	for in, want := range valid {
		if got, err := NormalizeMAC(in); err != nil || got != want {
			t.Errorf("NormalizeMAC(%q) = %q, %v; want %q", in, got, err, want)
		}
	}

	for _, in := range []string{"", "00:1a", "zz:zz:zz", "00:11:22:33:44:55:66"} {
		if _, err := NormalizeMAC(in); err == nil {
			t.Errorf("NormalizeMAC(%q) should fail", in)
		}
	}
}

func TestLookupMAC(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/001A2B3C4D5E") {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		w.Write([]byte(`{"success":true,"found":true,"macPrefix":"001A2B","company":"ACME"}`))
	}))
	defer srv.Close()

	MACLookupURL = srv.URL + "/"

	info, err := LookupMAC(context.Background(), "00:1a:2b:3c:4d:5e")
	if err != nil {
		t.Fatal(err)
	}

	if !info.Found || info.Company != "ACME" {
		t.Errorf("unexpected info %+v", info)
	}
}

func TestHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"error":"Rate limit exceeded"}`))
	}))
	defer srv.Close()

	IPInfoURL = srv.URL

	_, err := LookupIP(context.Background())
	if err == nil || !strings.Contains(err.Error(), "Rate limit exceeded (HTTP 429)") {
		t.Fatalf("unexpected error: %v", err)
	}
}
