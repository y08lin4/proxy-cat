package service

import (
	"strings"
	"testing"
)

func TestValidateSubscriptionURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid HTTPS URL",
			url:     "https://example.com/sub",
			wantErr: false,
		},
		{
			name:    "file scheme should fail",
			url:     "file:///etc/passwd",
			wantErr: true,
			errMsg:  "unsupported URL scheme",
		},
		{
			name:    "loopback 127.0.0.1 should fail",
			url:     "http://127.0.0.1:9090",
			wantErr: true,
			errMsg:  "loopback",
		},
		{
			name:    "private 192.168.x.x should fail",
			url:     "http://192.168.1.1/sub",
			wantErr: true,
			errMsg:  "private network",
		},
		{
			name:    "link-local 169.254.x.x should fail",
			url:     "http://169.254.169.254/metadata",
			wantErr: true,
			errMsg:  "link-local",
		},
		{
			name:    "localhost should fail",
			url:     "http://localhost:8080/api",
			wantErr: true,
			errMsg:  "loopback",
		},
		{
			name:    "0.0.0.0 should fail",
			url:     "http://0.0.0.0:8080/api",
			wantErr: true,
			errMsg:  "unspecified",
		},
		{
			name:    "private 10.x.x.x should fail",
			url:     "http://10.0.0.1/admin",
			wantErr: true,
			errMsg:  "private network",
		},
		{
			name:    "private 172.16.x.x should fail",
			url:     "http://172.16.0.1/config",
			wantErr: true,
			errMsg:  "private network",
		},
		{
			name:    "empty url should fail",
			url:     "",
			wantErr: true,
			errMsg:  "unsupported URL scheme",
		},
		{
			name:    "garbage url should fail",
			url:     "not-a-url:::",
			wantErr: true,
			errMsg:  "unsupported URL scheme",
		},
		{
			name:    "no host should fail",
			url:     "https:///path",
			wantErr: true,
			errMsg:  "no host",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSubscriptionURL(tt.url)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error for %q, got nil", tt.url)
				} else if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("expected error containing %q, got %q", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error for %q: %v", tt.url, err)
				}
			}
		})
	}
}
