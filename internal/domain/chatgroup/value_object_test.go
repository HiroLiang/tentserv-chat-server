package chatgroup

import (
	"errors"
	"testing"
)

func TestParseGroupType(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    GroupType
		wantErr error
	}{
		{name: "lowercase", input: "direct", want: Direct},
		{name: "uppercase", input: "DIRECT", want: Direct},
		{name: "mixed case", input: "GrOup", want: Group},
		{name: "trimmed", input: " channel ", want: Channel},
		{name: "invalid", input: "domain", wantErr: ErrInvalidGroupType},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseGroupType(tt.input)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("expected error %v, got %v", tt.wantErr, err)
				}
			} else {
				if got != tt.want {
					t.Fatalf("expected %v, got %v", tt.want, got)
				}
			}
		})
	}
}
