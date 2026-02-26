package participant

import (
	"errors"
	"testing"
)

func TestParseParticipantType(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    ParticipantType
		wantErr error
	}{
		{name: "lowercase", input: "user", want: UserType},
		{name: "uppercase", input: "USER", want: UserType},
		{name: "mixed case", input: "AgEnT", want: AgentType},
		{name: "trimmed", input: " system ", want: SystemType},
		{name: "invalid", input: "domain", wantErr: ErrInvalidParticipantType},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseParticipantType(tt.input)
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
