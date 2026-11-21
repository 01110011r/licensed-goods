package catalogue

import (
	"errors"
	"strings"
	"testing"
)

func TestValidateNewProduct(t *testing.T) {
	valid := NewProduct{
		SellerID:   "s1",
		Name:       "Test vodka",
		CategoryID: "c8",
		IKPU:       "22042100999999999",
	}

	tests := []struct {
		name       string
		mutate     func(*NewProduct)
		wantErr    bool
		wantStatus string
	}{
		{name: "valid defaults to draft", mutate: func(*NewProduct) {}, wantStatus: "draft"},
		{name: "trims whitespace", mutate: func(p *NewProduct) { p.Name = "  Vodka  "; p.IKPU = " 22042100999999999 " }, wantStatus: "draft"},
		{name: "explicit active", mutate: func(p *NewProduct) { p.Status = "active" }, wantStatus: "active"},
		{name: "missing seller", mutate: func(p *NewProduct) { p.SellerID = "" }, wantErr: true},
		{name: "missing name", mutate: func(p *NewProduct) { p.Name = "   " }, wantErr: true},
		{name: "name too long", mutate: func(p *NewProduct) { p.Name = strings.Repeat("a", 201) }, wantErr: true},
		{name: "missing category", mutate: func(p *NewProduct) { p.CategoryID = "" }, wantErr: true},
		{name: "ikpu too short", mutate: func(p *NewProduct) { p.IKPU = "2204210099999999" }, wantErr: true},
		{name: "ikpu too long", mutate: func(p *NewProduct) { p.IKPU = "220421009999999990" }, wantErr: true},
		{name: "ikpu with letters", mutate: func(p *NewProduct) { p.IKPU = "2204210099999999A" }, wantErr: true},
		{name: "ikpu with spaces inside", mutate: func(p *NewProduct) { p.IKPU = "2204 100999999999" }, wantErr: true},
		{name: "unknown status", mutate: func(p *NewProduct) { p.Status = "published" }, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := valid
			tt.mutate(&input)

			got, err := validateNewProduct(input)

			if tt.wantErr {
				if !errors.Is(err, ErrInvalidProduct) {
					t.Fatalf("expected ErrInvalidProduct, got %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.Status != tt.wantStatus {
				t.Fatalf("status = %q, want %q", got.Status, tt.wantStatus)
			}
		})
	}
}
