package licensing

import (
	"errors"
	"testing"
	"time"
)

func TestValidateLicenceSubmission(t *testing.T) {
	tests := []struct {
		name    string
		input   LicenceSubmission
		wantErr bool
	}{
		{
			name: "valid submission",
			input: LicenceSubmission{
				SellerID:  "s1",
				GroupID:   "MED",
				Number:    "MED-2026-001",
				Issuer:    "Ministry",
				ValidFrom: "2026-11-21",
				ValidTo:   "2027-11-21",
			},
		},
		{
			name: "trims whitespace",
			input: LicenceSubmission{
				SellerID:  " s1 ",
				GroupID:   " MED ",
				Number:    " MED-2026-001 ",
				Issuer:    " Ministry ",
				ValidFrom: "2026-11-21",
			},
		},
		{
			name: "missing seller",
			input: LicenceSubmission{
				GroupID: "MED", Number: "N1",
				Issuer: "Ministry", ValidFrom: "2026-11-21",
			},
			wantErr: true,
		},
		{
			name: "missing group",
			input: LicenceSubmission{
				SellerID: "s1", Number: "N1",
				Issuer: "Ministry", ValidFrom: "2026-11-21",
			},
			wantErr: true,
		},
		{
			name: "missing number",
			input: LicenceSubmission{
				SellerID: "s1", GroupID: "MED",
				Issuer: "Ministry", ValidFrom: "2026-11-21",
			},
			wantErr: true,
		},
		{
			name: "missing issuer",
			input: LicenceSubmission{
				SellerID: "s1", GroupID: "MED",
				Number: "N1", ValidFrom: "2026-11-21",
			},
			wantErr: true,
		},
		{
			name: "missing start date",
			input: LicenceSubmission{
				SellerID: "s1", GroupID: "MED",
				Number: "N1", Issuer: "Ministry",
			},
			wantErr: true,
		},
		{
			name: "invalid start date",
			input: LicenceSubmission{
				SellerID: "s1", GroupID: "MED",
				Number: "N1", Issuer: "Ministry",
				ValidFrom: "21-11-2026",
			},
			wantErr: true,
		},
		{
			name: "invalid end date",
			input: LicenceSubmission{
				SellerID: "s1", GroupID: "MED",
				Number: "N1", Issuer: "Ministry",
				ValidFrom: "2026-11-21",
				ValidTo:   "not-a-date",
			},
			wantErr: true,
		},
		{
			name: "end date before start date",
			input: LicenceSubmission{
				SellerID: "s1", GroupID: "MED",
				Number: "N1", Issuer: "Ministry",
				ValidFrom: "2026-11-21",
				ValidTo:   "2026-11-20",
			},
			wantErr: true,
		},
		{
			name: "end date equals start date",
			input: LicenceSubmission{
				SellerID: "s1", GroupID: "MED",
				Number: "N1", Issuer: "Ministry",
				ValidFrom: "2026-11-21",
				ValidTo:   "2026-11-21",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, validFrom, validTo, err :=
				validateLicenceSubmission(tt.input)

			if tt.wantErr {
				if !errors.Is(err, ErrInvalidSubmission) {
					t.Fatalf("expected ErrInvalidSubmission, got %v", err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.name == "trims whitespace" {
				if got.SellerID != "s1" ||
					got.GroupID != "MED" ||
					got.Number != "MED-2026-001" ||
					got.Issuer != "Ministry" {
					t.Fatalf("input was not normalized: %+v", got)
				}
			}

			if validFrom.Format("2006-01-02") != "2026-11-21" {
				t.Fatalf("unexpected validFrom: %v", validFrom)
			}

			if tt.input.ValidTo == "" {
				if validTo != nil {
					t.Fatalf("expected nil validTo, got %v", validTo)
				}
			} else {
				end, ok := validTo.(time.Time)
				if !ok {
					t.Fatalf("expected time.Time validTo, got %T", validTo)
				}
				if end.Before(validFrom) {
					t.Fatalf("validTo must not precede validFrom")
				}
			}
		})
	}
}
