package chatmessage

import (
	"errors"
	"testing"
)

func TestParseMessageType(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    MessageType
		wantErr error
	}{
		{name: "lowercase", input: "text", want: Text},
		{name: "uppercase", input: "TEXT", want: Text},
		{name: "mixed case", input: "ImAgE", want: Image},
		{name: "trimmed", input: " file ", want: File},
		{name: "invalid", input: "domain", wantErr: ErrInvalidMessageType},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseMessageType(tt.input)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("expected error %v, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}
}
