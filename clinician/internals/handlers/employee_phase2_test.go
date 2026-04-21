package handlers

import (
	"database/sql"
	"testing"
)

func TestBuildFacilityLeaveReviewURL(t *testing.T) {
	tests := []struct {
		name         string
		status       string
		departmentID int
		year         int
		month        int
		week         int
		want         string
	}{
		{
			name:   "pending no scope returns base route",
			status: "pending",
			want:   "/leave/review?year=0&month=0&week=0",
		},
		{
			name:         "status and scope are preserved",
			status:       "all",
			departmentID: 12,
			year:         2026,
			month:        4,
			week:         16,
			want:         "/leave/review?status=all&department=12&year=2026&month=4&week=16",
		},
		{
			name:         "pending keeps scoped params",
			status:       "pending",
			departmentID: 3,
			year:         2026,
			want:         "/leave/review?department=3&year=2026&month=0&week=0",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := buildFacilityLeaveReviewURL(tc.status, tc.departmentID, tc.year, tc.month, tc.week)
			if got != tc.want {
				t.Fatalf("buildFacilityLeaveReviewURL() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestBuildFacilityLeaveReviewExportURL(t *testing.T) {
	tests := []struct {
		name         string
		format       string
		status       string
		departmentID int
		year         int
		month        int
		week         int
		want         string
	}{
		{
			name:   "default pending export",
			format: "csv",
			status: "pending",
			want:   "/leave/review/export?format=csv&year=0&month=0&week=0",
		},
		{
			name:         "pdf export with full scope",
			format:       "pdf",
			status:       "on_leave",
			departmentID: 8,
			year:         2026,
			month:        4,
			week:         16,
			want:         "/leave/review/export?format=pdf&status=on_leave&department=8&year=2026&month=4&week=16",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := buildFacilityLeaveReviewExportURL(tc.format, tc.status, tc.departmentID, tc.year, tc.month, tc.week)
			if got != tc.want {
				t.Fatalf("buildFacilityLeaveReviewExportURL() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestSanitizeEmployeeListURL(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  string
	}{
		{name: "empty defaults to employee list", value: "", want: "/employee/list"},
		{name: "employee list accepted", value: "/employee/list?tab=on_leave&department=2", want: "/employee/list?tab=on_leave&department=2"},
		{name: "staff leave roster accepted", value: "/employee/leave/staffonleave", want: "/employee/leave/staffonleave"},
		{name: "external path rejected", value: "/reports/analysis", want: "/employee/list"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := sanitizeEmployeeListURL(tc.value)
			if got != tc.want {
				t.Fatalf("sanitizeEmployeeListURL(%q) = %q, want %q", tc.value, got, tc.want)
			}
		})
	}
}

func TestExportLeaveStatusLegacyExpired(t *testing.T) {
	got := exportLeaveStatus(nullableString("Expired"))
	if got != "Completed" {
		t.Fatalf("exportLeaveStatus(Expired) = %q, want %q", got, "Completed")
	}
}

func nullableString(value string) sql.NullString {
	if value == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: value, Valid: true}
}
