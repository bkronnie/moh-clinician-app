package handlers

import (
	"testing"

	"github.com/moh/clinician/internals/utilities"
)

func TestCanAccessReportSubmissions(t *testing.T) {
	tests := []struct {
		name string
		role string
		want bool
	}{
		{name: "national admin", role: utilities.RoleNationalAdmin, want: true},
		{name: "facility admin", role: utilities.RoleFacilityAdmin, want: true},
		{name: "staff", role: utilities.RoleStaff, want: false},
		{name: "unknown", role: "Visitor", want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := canAccessReportSubmissions(tc.role)
			if got != tc.want {
				t.Fatalf("canAccessReportSubmissions(%q) = %v, want %v", tc.role, got, tc.want)
			}
		})
	}
}

func TestResolveReportSubmissionsRoute(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		isNational bool
		wantFlag   int
		wantDept   bool
		wantOK     bool
	}{
		{
			name:       "national facilities route",
			path:       "/reports/submissions",
			isNational: true,
			wantFlag:   1,
			wantDept:   false,
			wantOK:     true,
		},
		{
			name:       "facility departments route",
			path:       "/reports/submissions",
			isNational: false,
			wantFlag:   2,
			wantDept:   true,
			wantOK:     true,
		},
		{
			name:       "explicit departments route",
			path:       "/reports/submissions/departments",
			isNational: true,
			wantFlag:   2,
			wantDept:   true,
			wantOK:     true,
		},
		{
			name:       "invalid route",
			path:       "/reports/submissions/unknown",
			isNational: true,
			wantFlag:   0,
			wantDept:   false,
			wantOK:     false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			flag, dept, ok := resolveReportSubmissionsRoute(tc.path, tc.isNational)
			if flag != tc.wantFlag || dept != tc.wantDept || ok != tc.wantOK {
				t.Fatalf("resolveReportSubmissionsRoute(%q, %v) = (%d, %v, %v), want (%d, %v, %v)", tc.path, tc.isNational, flag, dept, ok, tc.wantFlag, tc.wantDept, tc.wantOK)
			}
		})
	}
}

func TestResolveSubmissionsScopeFacilityID(t *testing.T) {
	tests := []struct {
		name              string
		isNational        bool
		currentHospitalID int64
		facilityParam     string
		wantFacilityID    int64
		wantErr           bool
	}{
		{
			name:              "facility admin uses session facility",
			isNational:        false,
			currentHospitalID: 19,
			facilityParam:     "",
			wantFacilityID:    19,
			wantErr:           false,
		},
		{
			name:              "facility admin without assignment denied",
			isNational:        false,
			currentHospitalID: 0,
			facilityParam:     "",
			wantFacilityID:    0,
			wantErr:           true,
		},
		{
			name:              "national requires facility",
			isNational:        true,
			currentHospitalID: 0,
			facilityParam:     "",
			wantFacilityID:    0,
			wantErr:           true,
		},
		{
			name:              "national invalid facility",
			isNational:        true,
			currentHospitalID: 0,
			facilityParam:     "abc",
			wantFacilityID:    0,
			wantErr:           true,
		},
		{
			name:              "national valid facility",
			isNational:        true,
			currentHospitalID: 0,
			facilityParam:     "7",
			wantFacilityID:    7,
			wantErr:           false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotFacilityID, err := resolveSubmissionsScopeFacilityID(tc.isNational, tc.currentHospitalID, tc.facilityParam)
			if (err != nil) != tc.wantErr {
				t.Fatalf("resolveSubmissionsScopeFacilityID(%v, %d, %q) error = %v, wantErr %v", tc.isNational, tc.currentHospitalID, tc.facilityParam, err, tc.wantErr)
			}
			if gotFacilityID != tc.wantFacilityID {
				t.Fatalf("resolveSubmissionsScopeFacilityID(%v, %d, %q) facility = %d, want %d", tc.isNational, tc.currentHospitalID, tc.facilityParam, gotFacilityID, tc.wantFacilityID)
			}
		})
	}
}
