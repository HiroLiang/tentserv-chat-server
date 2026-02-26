package chatmember

import (
	"errors"
	"testing"
)

func TestParseRole(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    Role
		wantErr error
	}{
		{name: "lowercase", input: "owner", want: Owner},
		{name: "uppercase", input: "OWNER", want: Owner},
		{name: "mixed case", input: "AdMiN", want: Admin},
		{name: "trimmed", input: " guest ", want: Guest},
		{name: "invalid", input: "domain", wantErr: ErrInvalidRole},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseRole(tt.input)
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
