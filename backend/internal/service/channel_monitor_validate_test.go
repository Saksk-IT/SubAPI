//go:build unit

package service

import (
	"errors"
	"testing"
)

func TestValidateEndpoint_AllowsHTTPAndHTTPSPublicOrigins(t *testing.T) {
	tests := []string{
		"http://203.0.113.10:10151/",
		"https://203.0.113.10",
	}

	for _, endpoint := range tests {
		t.Run(endpoint, func(t *testing.T) {
			if err := validateEndpoint(endpoint); err != nil {
				t.Fatalf("validateEndpoint(%q) returned error: %v", endpoint, err)
			}
		})
	}
}

func TestValidateEndpoint_RejectsUnsupportedScheme(t *testing.T) {
	err := validateEndpoint("ftp://203.0.113.10")
	if !errors.Is(err, ErrChannelMonitorEndpointScheme) {
		t.Fatalf("expected ErrChannelMonitorEndpointScheme, got %v", err)
	}
}

func TestValidateEndpoint_RejectsHTTPPrivateHost(t *testing.T) {
	err := validateEndpoint("http://127.0.0.1:10151")
	if !errors.Is(err, ErrChannelMonitorEndpointPrivate) {
		t.Fatalf("expected ErrChannelMonitorEndpointPrivate, got %v", err)
	}
}
