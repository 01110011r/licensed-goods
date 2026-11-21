package review

import (
	"errors"
	"testing"
)

func TestValidateDecision(t *testing.T) {
	tests := []struct {
		name       string
		decision   string
		reason     string
		wantReason string
		wantErr    error
	}{
		{
			name:     "approve",
			decision: "approved",
		},
		{
			name:       "approve ignores rejection reason",
			decision:   "approved",
			reason:     "not applicable",
			wantReason: "",
		},
		{
			name:       "reject with reason",
			decision:   "rejected",
			reason:     "  Invalid documents  ",
			wantReason: "Invalid documents",
		},
		{
			name:     "reject without reason",
			decision: "rejected",
			wantErr:  ErrReasonRequired,
		},
		{
			name:     "reject with whitespace reason",
			decision: "rejected",
			reason:   "   ",
			wantErr:  ErrReasonRequired,
		},
		{
			name:     "invalid decision",
			decision: "pending",
			wantErr:  ErrInvalidDecision,
		},
		{
			name:     "empty decision",
			decision: "",
			wantErr:  ErrInvalidDecision,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotReason, err := validateDecision(tt.decision, tt.reason)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected error %v, got %v", tt.wantErr, err)
			}

			if err == nil && gotReason != tt.wantReason {
				t.Fatalf("expected reason %q, got %q", tt.wantReason, gotReason)
			}
		})
	}
}
