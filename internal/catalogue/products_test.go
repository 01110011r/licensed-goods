package catalogue

import (
	"testing"
)

func TestApplyLicenceRequirements(t *testing.T) {
	tests := []struct {
		name          string
		sellerID      string
		groupIDs      []string
		groupNames    map[string]string
		validLicences map[string]bool
		wantOnHold    bool
		wantReasons   int
	}{
		{
			name:          "no required groups",
			sellerID:      "s1",
			groupNames:    map[string]string{},
			validLicences: map[string]bool{},
		},
		{
			name:     "all required licences valid",
			sellerID: "s1",
			groupIDs: []string{"MED", "ALC"},
			groupNames: map[string]string{
				"MED": "Medicine",
				"ALC": "Alcohol",
			},
			validLicences: map[string]bool{
				"s1\x00MED": true,
				"s1\x00ALC": true,
			},
		},
		{
			name:       "missing one of multiple licences",
			sellerID:   "s1",
			groupIDs:   []string{"MED", "ALC"},
			groupNames: map[string]string{},
			validLicences: map[string]bool{
				"s1\x00MED": true,
			},
			wantOnHold:  true,
			wantReasons: 1,
		},
		{
			name:          "all licences missing",
			sellerID:      "s1",
			groupIDs:      []string{"MED", "ALC"},
			groupNames:    map[string]string{},
			validLicences: map[string]bool{},
			wantOnHold:    true,
			wantReasons:   2,
		},
		{
			name:       "licence belonging to another seller does not count",
			sellerID:   "s1",
			groupIDs:   []string{"MED"},
			groupNames: map[string]string{},
			validLicences: map[string]bool{
				"s2\x00MED": true,
			},
			wantOnHold:  true,
			wantReasons: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			view := ProductView{
				SellerID: tt.sellerID,
			}

			applyLicenceRequirements(
				&view,
				tt.groupIDs,
				tt.groupNames,
				tt.validLicences,
			)

			if view.OnHold != tt.wantOnHold {
				t.Errorf("OnHold = %v, want %v", view.OnHold, tt.wantOnHold)
			}

			if len(view.HoldReasons) != tt.wantReasons {
				t.Errorf(
					"HoldReasons count = %d, want %d",
					len(view.HoldReasons),
					tt.wantReasons,
				)
			}

			if len(view.RequiredGroups) != len(tt.groupIDs) {
				t.Errorf(
					"RequiredGroups count = %d, want %d",
					len(view.RequiredGroups),
					len(tt.groupIDs),
				)
			}
		})
	}
}

func TestBuyerCatalogueVisibility(t *testing.T) {
	products := []ProductView{
		{ID: "p1", Status: "active", OnHold: false},
		{ID: "p2", Status: "active", OnHold: true},
		{ID: "p3", Status: "draft", OnHold: false},
		{ID: "p4", Status: "archived", OnHold: false},
	}

	visible := filterBuyerCatalogue(products)

	if len(visible) != 1 {
		t.Fatalf("expected 1 visible product, got %d", len(visible))
	}

	if visible[0].ID != "p1" {
		t.Errorf("expected p1, got %s", visible[0].ID)
	}
}
