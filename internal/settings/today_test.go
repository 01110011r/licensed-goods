package settings

import (
	"errors"
	"testing"
)

func TestValidateToday(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{name: "valid date", value: "2026-11-21"},
		{name: "leap day", value: "2024-02-29"},
		{name: "invalid leap day", value: "2025-02-29", wantErr: true},
		{name: "invalid month", value: "2026-13-01", wantErr: true},
		{name: "invalid day", value: "2026-04-31", wantErr: true},
		{name: "missing leading zero", value: "2026-1-01", wantErr: true},
		{name: "wrong format", value: "21-11-2026", wantErr: true},
		{name: "timestamp instead of date", value: "2026-11-21T10:00:00Z", wantErr: true},
		{name: "empty value", value: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateToday(tt.value)

			if tt.wantErr {
				if !errors.Is(err, ErrInvalidDate) {
					t.Fatalf("expected ErrInvalidDate, got %v", err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
