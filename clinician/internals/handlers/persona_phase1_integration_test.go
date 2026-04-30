package handlers

import (
	"strings"
	"testing"

	"github.com/moh/clinician/internals/utilities"
)

func TestPersonaLandingNavDrilldownAndExportScope(t *testing.T) {
	t.Run("national admin", func(t *testing.T) {
		view := HomeViewModel{
			Role:               utilities.RoleNationalAdmin,
			SelectedFacility:   11,
			SelectedDepartment: 7,
			SelectedEmployee:   41,
			SelectedYear:       2026,
			SelectedMonth:      4,
			SelectedWeek:       16,
			KPIs: []DashboardKPIView{
				{Title: "Reports Submitted"},
				{Title: "Pending Approval"},
				{Title: "Clinicians in Scope"},
			},
		}

		standardizeHomeViewModel(&view)

		reportsSubmitted := findKPIByTitle(view.KPIs, "Reports Submitted")
		if reportsSubmitted == nil {
			t.Fatalf("missing KPI: Reports Submitted")
		}
		expectURLContainsAll(t, reportsSubmitted.Link,
			"/reports/analysis",
			"facility=11",
			"department=7",
			"year=2026",
			"month=4",
			"week=16",
		)

		pendingApproval := findKPIByTitle(view.KPIs, "Pending Approval")
		if pendingApproval == nil {
			t.Fatalf("missing KPI: Pending Approval")
		}
		expectURLContainsAll(t, pendingApproval.Link,
			"/reports/analysis",
			"facility=11",
			"department=7",
			"year=2026",
			"month=4",
			"week=16",
			"status=pending",
		)

		clinicians := findKPIByTitle(view.KPIs, "Clinicians in Scope")
		if clinicians == nil {
			t.Fatalf("missing KPI: Clinicians in Scope")
		}
		expectURLContainsAll(t, clinicians.Link,
			"/?",
			"tab=clinicians",
			"facility=11",
			"department=7",
			"employee=41",
			"year=2026",
			"month=4",
			"week=16",
		)

		exportURL := buildReportSubmissionsExportURLWithView("csv", 11, 7, 2026, 4, 16, "pending", "facility")
		expectURLContainsAll(t, exportURL,
			"/reports/analysis/export",
			"format=csv",
			"facility=11",
			"department=7",
			"year=2026",
			"month=4",
			"week=16",
			"status=pending",
			"view=facility",
		)
	})

	t.Run("facility admin", func(t *testing.T) {
		view := HomeViewModel{
			Role:               utilities.RoleFacilityAdmin,
			SelectedDepartment: 5,
			SelectedYear:       2026,
			SelectedMonth:      4,
			SelectedWeek:       16,
			KPIs: []DashboardKPIView{
				{Title: "Pending Leave Requests"},
				{Title: "Pending Report Approvals"},
			},
		}

		standardizeHomeViewModel(&view)

		pendingLeave := findKPIByTitle(view.KPIs, "Pending Leave Requests")
		if pendingLeave == nil {
			t.Fatalf("missing KPI: Pending Leave Requests")
		}
		expectURLContainsAll(t, pendingLeave.Link,
			"/leave/review",
			"department=5",
			"year=2026",
			"month=4",
			"week=16",
		)

		pendingReports := findKPIByTitle(view.KPIs, "Pending Report Approvals")
		if pendingReports == nil {
			t.Fatalf("missing KPI: Pending Report Approvals")
		}
		expectURLContainsAll(t, pendingReports.Link,
			"/reports/analysis",
			"department=5",
			"year=2026",
			"month=4",
			"week=16",
			"status=pending",
		)

		leaveExport := buildFacilityLeaveReviewExportURL("pdf", "pending", 5, 2026, 4, 16)
		expectURLContainsAll(t, leaveExport,
			"/leave/review/export",
			"format=pdf",
			"department=5",
			"year=2026",
			"month=4",
			"week=16",
		)
	})

	t.Run("staff", func(t *testing.T) {
		view := HomeViewModel{
			Role:          utilities.RoleStaff,
			SelectedYear:  2026,
			SelectedMonth: 4,
			SelectedWeek:  16,
			KPIs: []DashboardKPIView{
				{Title: "Total Reports"},
				{Title: "Submitted"},
				{Title: "Approved"},
				{Title: "Declined"},
			},
		}

		standardizeHomeViewModel(&view)

		totalReports := findKPIByTitle(view.KPIs, "Total Reports")
		if totalReports == nil {
			t.Fatalf("missing KPI: Total Reports")
		}
		expectURLContainsAll(t, totalReports.Link,
			"/reports/analysis",
			"year=2026",
			"month=4",
			"week=16",
		)

		submitted := findKPIByTitle(view.KPIs, "Submitted")
		if submitted == nil {
			t.Fatalf("missing KPI: Submitted")
		}
		expectURLContainsAll(t, submitted.Link,
			"/reports/analysis",
			"year=2026",
			"month=4",
			"week=16",
			"status=submitted",
		)

		staffExport := buildReportHistoryExportURL("csv", "declined", 2026, 4, 16)
		expectURLContainsAll(t, staffExport,
			"/reports/analysis/export",
			"format=csv",
			"year=2026",
			"month=4",
			"week=16",
			"status=declined",
		)
	})
}

func findKPIByTitle(kpis []DashboardKPIView, title string) *DashboardKPIView {
	for i := range kpis {
		if kpis[i].Title == title {
			return &kpis[i]
		}
	}
	return nil
}

func expectURLContainsAll(t *testing.T, rawURL string, parts ...string) {
	t.Helper()
	for _, part := range parts {
		if !strings.Contains(rawURL, part) {
			t.Fatalf("url %q does not contain %q", rawURL, part)
		}
	}
}
