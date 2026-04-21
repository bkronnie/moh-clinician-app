package models

import "testing"

func TestEvaluateCanonicalRightsDrift(t *testing.T) {
	tests := []struct {
		name    string
		actual  []string
		wantLen int
	}{
		{
			name:    "canonical set no drift",
			actual:  []string{"National Admin", "Facility Admin", "Staff"},
			wantLen: 0,
		},
		{
			name:    "missing staff",
			actual:  []string{"National Admin", "Facility Admin"},
			wantLen: 2,
		},
		{
			name:    "unexpected legacy roles",
			actual:  []string{"National Admin", "Facility Admin", "Staff", "admin", "approver"},
			wantLen: 2,
		},
		{
			name:    "case and spacing tolerated",
			actual:  []string{" national admin ", "FACILITY ADMIN", "staff"},
			wantLen: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := evaluateCanonicalRightsDrift(tc.actual)
			if len(got) != tc.wantLen {
				t.Fatalf("evaluateCanonicalRightsDrift() len = %d, want %d; details=%v", len(got), tc.wantLen, got)
			}
		})
	}
}
