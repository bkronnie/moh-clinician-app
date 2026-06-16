package handlers

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/alexedwards/scs/v2"
	"github.com/gin-gonic/gin"
	"github.com/moh/clinician/internals/models"
	"github.com/moh/clinician/internals/utilities"
)

type ReportSubmissionsView struct {
	Role                   string
	ViewMode               string
	Mode                   string
	ShowWeekSummary        bool
	ShowDayBreakdown       bool
	ScopeTitle             string
	ScopeSubtitle          string
	FilterSummary          string
	SelectedFacility       int
	FacilityOptions        []models.DashboardFilterOption
	SelectedDepartment     int
	DepartmentOptions      []models.DashboardFilterOption
	SelectedStatus         string
	SelectedYear           int
	SelectedMonth          int
	SelectedWeek           int
	SelectedWeekLabel      string
	AvailableYears         []int
	AvailableMonths        []models.DashboardFilterOption
	AvailableWeeks         []models.ClinicianWeekOption
	Rows                   []*models.ReportSubmissionListRow
	FacilityRows           []*models.FacilitySubmissionSummaryRow
	NationalFacilityGroups []*ReportSubmissionNationalFacilityGroup
	WeekRows               []*ReportSubmissionWeekSummaryRow
	StaffWeekRows          []*ReportSubmissionStaffWeekSummaryRow
	WeekDayRows            []*models.WeekDayRow
	CurrentURL             string
	AllURL                 string
	SubmittedURL           string
	PendingURL             string
	ApprovedURL            string
	DeclinedURL            string
	DraftURL               string
	ExportCSVURL           string
	ExportPDFURL           string
	CanApprove             bool
	CanView                bool
	CanSubmitAll           bool
	SubmitAllURL           string
	PendingCount           int
	ClearFiltersURL        string
	FacilityModeURL        string
	BackToWeeksURL         string
	Page                   int
	PageSize               int
	TotalRows              int
	TotalPages             int
	PrevPageURL            string
	NextPageURL            string
	BatchDeclined          bool
	CanSubmitToNational    bool
	SubmitBlockedReason    string
	DraftCount             int
	CanApproveAll          bool
}

type ReportSubmissionWeekSummaryRow struct {
	Summary           *models.FacilitySubmissionSummaryRow
	DrilldownURL      string
	SubmissionPct     int
	ExpectedCount     int
	NotSubmittedCount int
	StaffRows         []*models.ReportSubmissionListRow
	DayGroups         []*ReportSubmissionNationalDayGroup
}

type ReportSubmissionNationalFacilityGroup struct {
	FacilityID   int
	FacilityName string
	WeekRows     []*ReportSubmissionWeekSummaryRow
}

type ReportSubmissionNationalDayGroup struct {
	Date           time.Time
	DayName        string
	SubmittedCount int
	ApprovedCount  int
	DeclinedCount  int
	DraftCount     int
	StaffRows      []*models.ReportSubmissionListRow
}

type ReportSubmissionStaffWeekSummaryRow struct {
	WeekStart         time.Time
	WeekStop          time.Time
	SubmittedCount    int
	ApprovedCount     int
	DeclinedCount     int
	NotSubmittedCount int
	SubmissionPct     int
	ApprovalStatus    string
	DayRows           []*ReportSubmissionStaffDayRow
}

type ReportSubmissionStaffDayRow struct {
	Date       time.Time
	Status     string
	Report     *models.ReportSubmissionListRow
	Actionable bool
}

func reportSubmissionDisplayStatus(row *models.ReportSubmissionListRow) string {
	if row == nil || row.Missing {
		return "Not Submitted"
	}
	reportStatus := row.ReportStatus.String
	submitStatus := row.SubmitStatus.String
	switch {
	case reportStatus == "Approved":
		return "Approved"
	case reportStatus == "Rejected" || reportStatus == "Declined":
		return "Declined"
	case submitStatus == "Submitted":
		return "Submitted"
	default:
		return "Draft"
	}
}

func buildStaffWeekSummaryRows(rows []*models.ReportSubmissionListRow, selectedStatus string) []*ReportSubmissionStaffWeekSummaryRow {
	type weekBucket struct {
		summary *ReportSubmissionStaffWeekSummaryRow
		byDate  map[string]*models.ReportSubmissionListRow
	}
	buckets := map[string]*weekBucket{}
	for _, row := range rows {
		if row == nil || !row.WeekStart.Valid {
			continue
		}
		day := row.WeekStart.Time
		weekday := int(day.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		weekStart := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, day.Location()).AddDate(0, 0, -(weekday - 1))
		key := weekStart.Format("2006-01-02")
		bucket := buckets[key]
		if bucket == nil {
			bucket = &weekBucket{
				summary: &ReportSubmissionStaffWeekSummaryRow{
					WeekStart: weekStart,
					WeekStop:  weekStart.AddDate(0, 0, 6),
				},
				byDate: map[string]*models.ReportSubmissionListRow{},
			}
			buckets[key] = bucket
		}
		bucket.byDate[day.Format("2006-01-02")] = row
	}

	items := make([]*ReportSubmissionStaffWeekSummaryRow, 0, len(buckets))
	for _, bucket := range buckets {
		summary := bucket.summary
		for dayOffset := 0; dayOffset < 7; dayOffset++ {
			day := summary.WeekStart.AddDate(0, 0, dayOffset)
			row := bucket.byDate[day.Format("2006-01-02")]
			status := reportSubmissionDisplayStatus(row)
			dayRow := &ReportSubmissionStaffDayRow{Date: day, Status: status, Report: row}
			if row != nil {
				dayRow.Actionable = row.Actionable
			}
			summary.DayRows = append(summary.DayRows, dayRow)
			switch status {
			case "Approved":
				summary.SubmittedCount++
				summary.ApprovedCount++
			case "Submitted", "Declined":
				summary.SubmittedCount++
				if status == "Declined" {
					summary.DeclinedCount++
				}
			case "Draft", "Not Submitted":
				summary.NotSubmittedCount++
			}
		}
		summary.SubmissionPct = summary.SubmittedCount * 100 / 7
		switch {
		case summary.ApprovedCount == 7:
			summary.ApprovalStatus = "Approved"
		case summary.DeclinedCount > 0:
			summary.ApprovalStatus = "Declined"
		case summary.SubmittedCount == 7:
			summary.ApprovalStatus = "Submitted"
		default:
			summary.ApprovalStatus = "Draft"
		}
		if selectedStatus != "all" {
			include := false
			switch selectedStatus {
			case "submitted", "pending":
				include = summary.ApprovalStatus == "Submitted"
			case "approved":
				include = summary.ApprovalStatus == "Approved"
			case "declined":
				include = summary.ApprovalStatus == "Declined"
			case "draft":
				include = summary.ApprovalStatus == "Draft"
			}
			if !include {
				continue
			}
		}
		items = append(items, summary)
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].WeekStart.After(items[j].WeekStart)
	})
	return items
}

func buildFacilityWeekSummaryRow(selectedFacility int, selectedDepartment int, selectedStatus string, row *models.FacilitySubmissionSummaryRow) *ReportSubmissionWeekSummaryRow {
	if row == nil {
		return nil
	}
	drilldownURL := buildFacilityWeekDrilldownURL(selectedFacility, selectedDepartment, selectedStatus, row)
	pct := 0
	expected := (row.OnDutyCount - row.OnLeaveCount) * 7
	if expected < 0 {
		expected = 0
	}
	notSubmitted := expected - row.SubmittedCount
	if notSubmitted < 0 {
		notSubmitted = 0
	}
	if expected > 0 {
		pct = row.SubmittedCount * 100 / expected
		if pct > 100 {
			pct = 100
		}
	}
	return &ReportSubmissionWeekSummaryRow{
		Summary:           row,
		DrilldownURL:      drilldownURL,
		SubmissionPct:     pct,
		ExpectedCount:     expected,
		NotSubmittedCount: notSubmitted,
	}
}

func loadNationalWeekStaffRows(c *gin.Context, db *sql.DB, facilityID int, departmentID int, selectedStatus string, weekStart time.Time, weekStop time.Time) ([]*models.ReportSubmissionListRow, error) {
	year, week := weekStart.ISOWeek()
	month := int(weekStart.Month())
	rows, _, err := models.GetReportSubmissionsPaged(c.Request.Context(), db, 0, 0, 0, facilityID, departmentID, selectedStatus, year, month, week, 0, 0)
	if err != nil {
		return nil, err
	}
	if selectedStatus == "all" {
		missingRows, err := models.GetMissingStaffForWeek(c.Request.Context(), db, facilityID, departmentID, weekStart, weekStop)
		if err != nil {
			return nil, err
		}
		rows = append(rows, missingRows...)
		sort.SliceStable(rows, func(i, j int) bool {
			leftName := strings.ToLower(rows[i].EmployeeName)
			rightName := strings.ToLower(rows[j].EmployeeName)
			if leftName == rightName {
				return rows[i].ReportID < rows[j].ReportID
			}
			return leftName < rightName
		})
	}
	return rows, nil
}

func buildNationalWeekDayGroups(weekStart time.Time, staffRows []*models.ReportSubmissionListRow) []*ReportSubmissionNationalDayGroup {
	groups := make([]*ReportSubmissionNationalDayGroup, 0, 7)
	byDate := map[string]*ReportSubmissionNationalDayGroup{}
	for dayOffset := 0; dayOffset < 7; dayOffset++ {
		date := weekStart.AddDate(0, 0, dayOffset)
		group := &ReportSubmissionNationalDayGroup{
			Date:    date,
			DayName: date.Format("Monday"),
		}
		groups = append(groups, group)
		byDate[date.Format("2006-01-02")] = group
	}

	for _, row := range staffRows {
		if row == nil || row.Missing || !row.WeekStart.Valid {
			continue
		}
		key := row.WeekStart.Time.Format("2006-01-02")
		group := byDate[key]
		if group == nil {
			continue
		}
		group.StaffRows = append(group.StaffRows, row)
		switch reportSubmissionDisplayStatus(row) {
		case "Approved":
			group.SubmittedCount++
			group.ApprovedCount++
		case "Declined":
			group.SubmittedCount++
			group.DeclinedCount++
		case "Submitted":
			group.SubmittedCount++
		case "Draft":
			group.DraftCount++
		}
	}

	for _, group := range groups {
		sort.SliceStable(group.StaffRows, func(i, j int) bool {
			leftName := strings.ToLower(group.StaffRows[i].EmployeeName)
			rightName := strings.ToLower(group.StaffRows[j].EmployeeName)
			if leftName == rightName {
				return group.StaffRows[i].ReportID < group.StaffRows[j].ReportID
			}
			return leftName < rightName
		})
	}

	return groups
}

func buildNationalFacilityGroups(c *gin.Context, db *sql.DB, facilityOptions []models.DashboardFilterOption, selectedFacility int, selectedDepartment int, selectedStatus string, selectedYear int, selectedMonth int, selectedWeek int) ([]*ReportSubmissionNationalFacilityGroup, error) {
	groups := []*ReportSubmissionNationalFacilityGroup{}
	for _, facilityOption := range facilityOptions {
		if selectedFacility > 0 && facilityOption.ID != selectedFacility {
			continue
		}
		weekRows, err := models.GetFacilityWeeklySubmissionSummaries(c.Request.Context(), db, facilityOption.ID, selectedDepartment, selectedStatus, selectedYear, selectedMonth, selectedWeek)
		if err != nil {
			return nil, err
		}
		if len(weekRows) == 0 {
			continue
		}
		group := &ReportSubmissionNationalFacilityGroup{
			FacilityID:   facilityOption.ID,
			FacilityName: facilityOption.Name,
		}
		for _, weekRow := range weekRows {
			summaryRow := buildFacilityWeekSummaryRow(facilityOption.ID, selectedDepartment, selectedStatus, weekRow)
			if summaryRow == nil {
				continue
			}
			if weekRow.WeekStart.Valid && weekRow.WeekStop.Valid {
				staffRows, err := loadNationalWeekStaffRows(c, db, facilityOption.ID, selectedDepartment, selectedStatus, weekRow.WeekStart.Time, weekRow.WeekStop.Time)
				if err != nil {
					return nil, err
				}
				summaryRow.StaffRows = staffRows
				summaryRow.DayGroups = buildNationalWeekDayGroups(weekRow.WeekStart.Time, staffRows)
			}
			group.WeekRows = append(group.WeekRows, summaryRow)
		}
		groups = append(groups, group)
	}
	return groups, nil
}

func buildReportSubmissionsView(c *gin.Context, db *sql.DB, sesDetails utilities.SessionDetails, paginate bool) (ReportSubmissionsView, error) {
	requestedFacility, hasFacility := parseOptionalIntQuery(c, "facility")
	requestedDepartment, hasDepartment := parseOptionalIntQuery(c, "department")
	requestedYear, hasYear := parseOptionalIntQuery(c, "year")
	requestedMonth, hasMonth := parseOptionalIntQuery(c, "month")
	requestedWeek, hasWeek := parseOptionalIntQuery(c, "week")
	selectedStatus := normalizeReportSubmissionStatus(c.Query("status"))
	viewMode := normalizeReportSubmissionViewMode(roleFromRights(sesDetails.Rights), c.Query("view"))
	requestedMode := normalizeReportSubmissionMode(roleFromRights(sesDetails.Rights), c.Query("mode"))
	// National admin has no draft concept — drafts only exist at the facility tier.
	if roleFromRights(sesDetails.Rights) == utilities.RoleNationalAdmin && selectedStatus == "draft" {
		selectedStatus = "all"
	}

	selectedYear, selectedMonth, selectedWeek, availableYears, availableMonths, availableWeeks, selectedWeekLabel, err := resolveReportSubmissionPeriod(c, db, requestedYear, hasYear, requestedMonth, hasMonth, requestedWeek, hasWeek)
	if err != nil {
		return ReportSubmissionsView{}, err
	}

	facilityOptions, err := models.GetDashboardFacilityOptions(c.Request.Context(), db)
	if err != nil {
		return ReportSubmissionsView{}, err
	}

	view := ReportSubmissionsView{
		Role:              roleFromRights(sesDetails.Rights),
		ViewMode:          viewMode,
		Mode:              requestedMode,
		ShowWeekSummary:   (roleFromRights(sesDetails.Rights) == utilities.RoleFacilityAdmin || (roleFromRights(sesDetails.Rights) == utilities.RoleNationalAdmin && viewMode == "facility") || roleFromRights(sesDetails.Rights) == utilities.RoleStaff) && requestedMode == "weeks",
		ShowDayBreakdown:  (roleFromRights(sesDetails.Rights) == utilities.RoleFacilityAdmin || (roleFromRights(sesDetails.Rights) == utilities.RoleNationalAdmin && viewMode == "facility")) && requestedMode == "days",
		SelectedStatus:    selectedStatus,
		SelectedYear:      selectedYear,
		SelectedMonth:     selectedMonth,
		SelectedWeek:      selectedWeek,
		SelectedWeekLabel: selectedWeekLabel,
		AvailableYears:    availableYears,
		AvailableMonths:   availableMonths,
		AvailableWeeks:    availableWeeks,
		CanApprove:        roleFromRights(sesDetails.Rights) == utilities.RoleFacilityAdmin || roleFromRights(sesDetails.Rights) == utilities.RoleNationalAdmin,
		CanView:           roleFromRights(sesDetails.Rights) != utilities.RoleStaff,
		CanSubmitAll:      roleFromRights(sesDetails.Rights) == utilities.RoleFacilityAdmin,
		Page:              1,
		PageSize:          50,
	}

	if paginate {
		if requestedPage, hasPage := parseOptionalIntQuery(c, "page"); hasPage && requestedPage > 0 {
			view.Page = requestedPage
		}
	}

	scopeFacilityID := int64(0)
	scopeEmployeeID := int64(0)
	if roleFromRights(sesDetails.Rights) == utilities.RoleStaff {
		scopeEmployeeID = sesDetails.EmpID
		view.ScopeTitle = "My Reports"
		view.ScopeSubtitle = "Review all of your saved, submitted, approved, and declined reports in one place using the same filtered reports screen used across the application."
		view.SelectedFacility = int(sesDetails.HFID)
		view.FacilityOptions = []models.DashboardFilterOption{{
			ID:   int(sesDetails.HFID),
			Name: sesDetails.HFName,
		}}
	} else if roleFromRights(sesDetails.Rights) == utilities.RoleFacilityAdmin {
		scopeFacilityID = sesDetails.HFID
		view.SelectedFacility = int(sesDetails.HFID)
		view.FacilityOptions = []models.DashboardFilterOption{{
			ID:   int(sesDetails.HFID),
			Name: sesDetails.HFName,
		}}
		view.ScopeTitle = "Facility Report Submissions"
		view.ScopeSubtitle = "Review all report submissions for your facility, inspect the saved data in read-only mode, approve pending submissions, and submit the full weekly batch at once."
	} else {
		if hasFacility {
			view.SelectedFacility = requestedFacility
		}
		view.FacilityOptions = facilityOptions
		view.ScopeTitle = "National Report Submissions"
		view.ScopeSubtitle = "Review report submissions across facilities. The default view is facility-level weekly submission status with drill-down to staff rows."
	}

	departmentOptions, err := models.GetReportSubmissionDepartmentOptions(c.Request.Context(), db, scopeFacilityID, view.SelectedFacility)
	if err != nil {
		return ReportSubmissionsView{}, err
	}
	view.DepartmentOptions = departmentOptions
	if roleFromRights(sesDetails.Rights) != utilities.RoleStaff && hasDepartment && containsFilterOption(departmentOptions, requestedDepartment) {
		view.SelectedDepartment = requestedDepartment
	}

	if roleFromRights(sesDetails.Rights) == utilities.RoleStaff && view.ShowWeekSummary {
		view.PageSize = 0
		view.Page = 1
		rows, _, err := models.GetReportSubmissionsPaged(c.Request.Context(), db, scopeFacilityID, scopeEmployeeID, sesDetails.EmpID, view.SelectedFacility, view.SelectedDepartment, "all", selectedYear, selectedMonth, selectedWeek, 0, 0)
		if err != nil {
			return ReportSubmissionsView{}, err
		}
		view.Rows = rows
		view.StaffWeekRows = buildStaffWeekSummaryRows(rows, selectedStatus)
		view.TotalRows = len(view.StaffWeekRows)
	} else if roleFromRights(sesDetails.Rights) == utilities.RoleNationalAdmin && viewMode == "facility" && view.ShowWeekSummary {
		view.PageSize = 0
		view.Page = 1
		groups, err := buildNationalFacilityGroups(c, db, view.FacilityOptions, view.SelectedFacility, view.SelectedDepartment, selectedStatus, selectedYear, selectedMonth, selectedWeek)
		if err != nil {
			return ReportSubmissionsView{}, err
		}
		view.NationalFacilityGroups = groups
		view.TotalRows = len(groups)
		for _, group := range groups {
			for _, row := range group.WeekRows {
				if row == nil || row.Summary == nil {
					continue
				}
				if row.Summary.PendingCount > 0 {
					view.PendingCount += row.Summary.PendingCount
				}
				if row.Summary.ApprovalStatus == "Declined" {
					view.BatchDeclined = true
				}
			}
		}
	} else if view.ShowWeekSummary {
		view.PageSize = 0
		view.Page = 1
		// Facility scope: facility admins always use their own facility;
		// national admins must have selected a facility to populate the
		// weekly accordion (template shows a prompt otherwise).
		summaryFacilityID := 0
		if roleFromRights(sesDetails.Rights) == utilities.RoleFacilityAdmin {
			summaryFacilityID = int(sesDetails.HFID)
		} else if view.SelectedFacility > 0 {
			summaryFacilityID = view.SelectedFacility
		}
		if summaryFacilityID > 0 {
			weekRows, err := models.GetFacilityWeeklySubmissionSummaries(c.Request.Context(), db, summaryFacilityID, view.SelectedDepartment, selectedStatus, selectedYear, selectedMonth, selectedWeek)
			if err != nil {
				return ReportSubmissionsView{}, err
			}
			view.FacilityRows = weekRows
			view.TotalRows = len(weekRows)
			for _, row := range weekRows {
				if row == nil {
					continue
				}
				if row.PendingCount > 0 {
					view.PendingCount += row.PendingCount
				}
				if row.ApprovalStatus == "Declined" {
					view.BatchDeclined = true
				}
				view.WeekRows = append(view.WeekRows, buildFacilityWeekSummaryRow(view.SelectedFacility, view.SelectedDepartment, selectedStatus, row))
			}
		}
	} else if view.ShowDayBreakdown {
		dayFacilityID := 0
		if roleFromRights(sesDetails.Rights) == utilities.RoleFacilityAdmin {
			dayFacilityID = int(sesDetails.HFID)
		} else if view.SelectedFacility > 0 {
			dayFacilityID = view.SelectedFacility
		}
		if dayFacilityID > 0 && selectedYear > 0 && selectedWeek > 0 {
			weekStart := isoWeekStart(selectedYear, selectedWeek)
			dayRows, err := models.GetWeekDailyBreakdown(c.Request.Context(), db, dayFacilityID, view.SelectedDepartment, weekStart)
			if err != nil {
				return ReportSubmissionsView{}, err
			}
			view.WeekDayRows = dayRows
			view.TotalRows = len(dayRows)
		}
	} else if roleFromRights(sesDetails.Rights) == utilities.RoleNationalAdmin && viewMode == "facility" {
		view.PageSize = 0
		view.Page = 1
		facilityRows, err := models.GetFacilitySubmissionSummaries(c.Request.Context(), db, view.SelectedFacility, selectedStatus, selectedYear, selectedMonth, selectedWeek)
		if err != nil {
			return ReportSubmissionsView{}, err
		}
		view.FacilityRows = facilityRows
		view.TotalRows = len(facilityRows)
		for _, row := range facilityRows {
			view.PendingCount += row.PendingCount
		}
	} else {
		limit := 0
		offset := 0
		if paginate {
			limit = view.PageSize
			offset = (view.Page - 1) * view.PageSize
		}

		// Facility admin drilling down to a specific week should also list staff
		// who never submitted a report for that week, marked "Not Submitted".
		// Disable pagination in that case so the missing rows always render.
		augmentWithMissing := roleFromRights(sesDetails.Rights) == utilities.RoleFacilityAdmin &&
			selectedYear > 0 && selectedWeek > 0 &&
			(selectedStatus == "all" || selectedStatus == "draft")
		if augmentWithMissing {
			limit = 0
			offset = 0
			view.PageSize = 0
			view.Page = 1
		}

		rows, totalRows, err := models.GetReportSubmissionsPaged(c.Request.Context(), db, scopeFacilityID, scopeEmployeeID, sesDetails.EmpID, view.SelectedFacility, view.SelectedDepartment, selectedStatus, selectedYear, selectedMonth, selectedWeek, limit, offset)
		if err != nil {
			return ReportSubmissionsView{}, err
		}
		view.Rows = rows
		view.TotalRows = totalRows

		for _, row := range rows {
			if row.SubmitStatus.Valid && row.SubmitStatus.String == "Submitted" && (!row.ReportStatus.Valid || !isFinalReportReviewStatus(row.ReportStatus.String)) {
				view.PendingCount++
			}
		}

		if augmentWithMissing {
			weekStart := isoWeekStart(selectedYear, selectedWeek)
			weekStop := weekStart.AddDate(0, 0, 6)
			missing, err := models.GetMissingStaffForWeek(c.Request.Context(), db, view.SelectedFacility, view.SelectedDepartment, weekStart, weekStop)
			if err != nil {
				return ReportSubmissionsView{}, err
			}
			if len(missing) > 0 {
				view.Rows = append(view.Rows, missing...)
				view.TotalRows += len(missing)
			}
		}

		// In facility admin staff-drilldown for a specific week, surface whether the
		// batch was previously submitted upward and declined so the UI can offer a
		// clear "Resubmit" affordance (the SQL guard still requires all rows approved).
		if roleFromRights(sesDetails.Rights) == utilities.RoleFacilityAdmin && selectedYear > 0 && selectedWeek > 0 {
			summaries, err := models.GetFacilityWeeklySubmissionSummaries(c.Request.Context(), db, int(sesDetails.HFID), view.SelectedDepartment, "all", selectedYear, selectedMonth, selectedWeek)
			if err == nil {
				for _, s := range summaries {
					if s != nil && s.ApprovalStatus == "Declined" {
						view.BatchDeclined = true
						break
					}
				}
			}
			readiness, err := models.GetFacilityWeekReadiness(c.Request.Context(), db, int(sesDetails.HFID), view.SelectedDepartment, selectedYear, selectedMonth, selectedWeek, sesDetails.EmpID)
			if err == nil {
				view.DraftCount = readiness.DraftCount
				// Step 1 (Approve All) becomes the single action that clears
				// drafts and pending submissions in scope (admin's own draft
				// excepted; promoted by Step 2's selfQuery).
				view.CanApproveAll = view.CanApprove && (readiness.DraftCount > 0 || readiness.PendingCount > 0)
				switch {
				case readiness.TotalCount == 0:
					view.CanSubmitToNational = false
					view.SubmitBlockedReason = "No reports captured for this week yet."
				case readiness.DraftCount > 0 && readiness.PendingCount > 0:
					view.CanSubmitToNational = false
					view.SubmitBlockedReason = fmt.Sprintf("%d draft and %d submitted-but-unapproved report(s) remain. Use 'Approve All Reports' first.", readiness.DraftCount, readiness.PendingCount)
				case readiness.DraftCount > 0:
					view.CanSubmitToNational = false
					view.SubmitBlockedReason = fmt.Sprintf("%d staff report(s) still in draft. Use 'Approve All Reports' to submit & approve them.", readiness.DraftCount)
				case readiness.PendingCount > 0:
					view.CanSubmitToNational = false
					view.SubmitBlockedReason = fmt.Sprintf("%d submitted report(s) awaiting facility approval. Use 'Approve All Reports' first.", readiness.PendingCount)
				default:
					view.CanSubmitToNational = true
				}
			}
		}
	}

	view.CurrentURL = buildReportSubmissionsURLWithModeAndPage(view.SelectedFacility, view.SelectedDepartment, view.SelectedYear, view.SelectedMonth, view.SelectedWeek, view.SelectedStatus, view.ViewMode, view.Mode, view.Page)
	view.AllURL = buildReportSubmissionsURLWithModeAndPage(view.SelectedFacility, view.SelectedDepartment, view.SelectedYear, view.SelectedMonth, view.SelectedWeek, "all", view.ViewMode, view.Mode, 1)
	view.SubmittedURL = buildReportSubmissionsURLWithModeAndPage(view.SelectedFacility, view.SelectedDepartment, view.SelectedYear, view.SelectedMonth, view.SelectedWeek, "submitted", view.ViewMode, view.Mode, 1)
	view.PendingURL = buildReportSubmissionsURLWithModeAndPage(view.SelectedFacility, view.SelectedDepartment, view.SelectedYear, view.SelectedMonth, view.SelectedWeek, "pending", view.ViewMode, view.Mode, 1)
	view.ApprovedURL = buildReportSubmissionsURLWithModeAndPage(view.SelectedFacility, view.SelectedDepartment, view.SelectedYear, view.SelectedMonth, view.SelectedWeek, "approved", view.ViewMode, view.Mode, 1)
	view.DeclinedURL = buildReportSubmissionsURLWithModeAndPage(view.SelectedFacility, view.SelectedDepartment, view.SelectedYear, view.SelectedMonth, view.SelectedWeek, "declined", view.ViewMode, view.Mode, 1)
	view.DraftURL = buildReportSubmissionsURLWithModeAndPage(view.SelectedFacility, view.SelectedDepartment, view.SelectedYear, view.SelectedMonth, view.SelectedWeek, "draft", view.ViewMode, view.Mode, 1)
	view.ExportCSVURL = buildReportSubmissionsExportURLWithView("csv", view.SelectedFacility, view.SelectedDepartment, view.SelectedYear, view.SelectedMonth, view.SelectedWeek, view.SelectedStatus, view.ViewMode)
	view.ExportPDFURL = buildReportSubmissionsExportURLWithView("pdf", view.SelectedFacility, view.SelectedDepartment, view.SelectedYear, view.SelectedMonth, view.SelectedWeek, view.SelectedStatus, view.ViewMode)
	view.ClearFiltersURL = buildReportSubmissionsURLWithModeAndPage(0, 0, 0, 0, 0, "all", view.ViewMode, view.Mode, 1)
	view.FacilityModeURL = buildReportSubmissionsURLWithViewAndPage(0, 0, view.SelectedYear, view.SelectedMonth, view.SelectedWeek, view.SelectedStatus, "facility", 1)
	view.BackToWeeksURL = buildReportSubmissionsURLWithModeAndPage(view.SelectedFacility, view.SelectedDepartment, view.SelectedYear, view.SelectedMonth, 0, view.SelectedStatus, view.ViewMode, "weeks", 1)
	view.SubmitAllURL = "/reports/analysis/submit-all"
	view.FilterSummary = reportSubmissionFilterSummary(roleFromRights(sesDetails.Rights), selectedStatus, selectedWeekLabel)

	if view.PageSize > 0 {
		if view.TotalRows <= 0 {
			view.TotalPages = 1
		} else {
			view.TotalPages = (view.TotalRows + view.PageSize - 1) / view.PageSize
		}
		if view.Page < 1 {
			view.Page = 1
		}
		if view.Page > view.TotalPages {
			view.Page = view.TotalPages
		}
		if view.Page > 1 {
			view.PrevPageURL = buildReportSubmissionsURLWithModeAndPage(view.SelectedFacility, view.SelectedDepartment, view.SelectedYear, view.SelectedMonth, view.SelectedWeek, view.SelectedStatus, view.ViewMode, view.Mode, view.Page-1)
		}
		if view.Page < view.TotalPages {
			view.NextPageURL = buildReportSubmissionsURLWithModeAndPage(view.SelectedFacility, view.SelectedDepartment, view.SelectedYear, view.SelectedMonth, view.SelectedWeek, view.SelectedStatus, view.ViewMode, view.Mode, view.Page+1)
		}
	}

	return view, nil
}

func HandlerReportsAnalysis(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {
	sessionManager.Put(c.Request.Context(), "reports_analysis_seen_at", time.Now().UTC().Format(time.RFC3339Nano))

	sessionData, ok := Get_Session_Data(c, db, sessionManager, nil).(utilities.TemplateData)
	if !ok {
		log.Println("Failed to retrieve session data as TemplateData")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to retrieve session data"})
		return
	}

	sesDetails, ok := sessionData.Ses.(utilities.SessionDetails)
	if !ok {
		log.Println("Failed to retrieve session details from TemplateData")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to retrieve session details"})
		return
	}

	view, err := buildReportSubmissionsView(c, db, sesDetails, true)
	if err != nil {
		log.Printf("Error building report submissions view: %v", err)
		c.String(http.StatusInternalServerError, "Error retrieving report submissions")
		return
	}

	sessionData.Form = view
	utilities.GenerateHTML(c, sessionData, "base", "report-analysis")
}

func HandlerReportsAnalysisExport(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {
	sessionData, ok := Get_Session_Data(c, db, sessionManager, nil).(utilities.TemplateData)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to retrieve session data"})
		return
	}

	sesDetails, ok := sessionData.Ses.(utilities.SessionDetails)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to retrieve session details"})
		return
	}

	view, err := buildReportSubmissionsView(c, db, sesDetails, false)
	if err != nil {
		log.Printf("Error exporting report submissions: %v", err)
		c.String(http.StatusInternalServerError, "Error exporting report submissions")
		return
	}

	format := c.DefaultQuery("format", "csv")
	if format != "csv" && format != "pdf" {
		c.String(http.StatusBadRequest, "Invalid export format")
		return
	}

	headers := []string{"Facility", "Department", "Employee", "Week", "Entered On", "Submitted On", "Status", "Attendance", "Patients Reviewed", "Procedures"}
	csvRows := [][]string{}
	pdfLines := []string{
		view.ScopeTitle,
		fmt.Sprintf("Period: %s", view.SelectedWeekLabel),
		"",
	}

	for index, row := range view.Rows {
		status := reportSubmissionStatusLabel(row.SubmitStatus, row.ReportStatus)
		weekLabel := exportWeekRange(row.WeekStart, row.WeekStop)
		enteredOn := exportDateTime(row.EnteredOn)
		submittedOn := exportDateTime(row.SubmittedOn)

		csvRows = append(csvRows, []string{
			row.FacilityName,
			row.DepartmentName,
			row.EmployeeName,
			weekLabel,
			enteredOn,
			submittedOn,
			status,
			strconv.Itoa(row.Attendance),
			strconv.Itoa(row.PatientsReviewed),
			strconv.Itoa(row.Procedures),
		})

		pdfLines = append(pdfLines,
			fmt.Sprintf("%d. %s - %s", index+1, row.EmployeeName, row.FacilityName),
			fmt.Sprintf("   Department: %s | Week: %s", row.DepartmentName, weekLabel),
			fmt.Sprintf("   Status: %s | Entered: %s | Submitted: %s", status, enteredOn, submittedOn),
			fmt.Sprintf("   Attendance: %d | Patients Reviewed: %d | Procedures: %d", row.Attendance, row.PatientsReviewed, row.Procedures),
			"",
		)
	}

	filenameBase := buildReportSubmissionsExportFilename(view.SelectedFacility, view.SelectedDepartment, view.SelectedYear, view.SelectedMonth, view.SelectedWeek, view.SelectedStatus)
	if format == "pdf" {
		writeSimplePDFDownload(c, filenameBase+".pdf", view.ScopeTitle, pdfLines)
		return
	}
	writeCSVDownload(c, filenameBase+".csv", headers, csvRows)
}

func HandlerReportSubmissionView(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {
	sessionData, ok := Get_Session_Data(c, db, sessionManager, nil).(utilities.TemplateData)
	if !ok {
		log.Println("Failed to retrieve session data as TemplateData")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to retrieve session data"})
		return
	}

	sesDetails, ok := sessionData.Ses.(utilities.SessionDetails)
	if !ok {
		log.Println("Failed to retrieve session details from TemplateData")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to retrieve session details"})
		return
	}

	reportID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid report ID")
		return
	}

	scopeFacilityID := int64(0)
	if roleFromRights(sesDetails.Rights) == utilities.RoleFacilityAdmin {
		scopeFacilityID = sesDetails.HFID
	}

	reviewRole := sesDetails.Rights
	report, err := models.GetReportSubmissionByIDForReview(c.Request.Context(), db, reportID, scopeFacilityID, reviewRole)
	if err != nil {
		if err == sql.ErrNoRows {
			c.String(http.StatusNotFound, "Report submission not found")
			return
		}
		log.Printf("Error retrieving report submission: %v", err)
		c.String(http.StatusInternalServerError, "Unable to retrieve report submission")
		return
	}
	if roleFromRights(sesDetails.Rights) != utilities.RoleStaff {
		isDraft := report.SubmitStatus.String != "Submitted" && report.ReportStatus.String != "Approved" && report.ReportStatus.String != "Rejected" && report.ReportStatus.String != "Declined"
		if isDraft {
			c.String(http.StatusNotFound, "Report submission not found")
			return
		}
	}

	employee, err := models.EmployeeByID(c, db, int(report.EmployeeID))
	if err != nil {
		c.String(http.StatusInternalServerError, "Unable to load employee details")
		return
	}

	department, err := models.DepartmentByID(c, db, int(report.DepartmentID))
	if err != nil {
		c.String(http.StatusInternalServerError, "Unable to load department details")
		return
	}

	facility, err := models.FacilityByID(c, db, int(report.FacilityID))
	if err != nil {
		c.String(http.StatusInternalServerError, "Unable to load facility details")
		return
	}

	labels := resolveClinicianEntryLabels(c.Request.Context(), db)
	subSections, subHideCore := resolveClinicianEntryConfig(c.Request.Context(), db, report.DepartmentID, labels)
	entryForm := ClinicianEntryView{
		EmployeeID:      report.EmployeeID,
		EmployeeName:    fmt.Sprintf("%s %s", employee.Fname.String, employee.Lname.String),
		DepartmentID:    report.DepartmentID,
		DepartmentName:  department.DepartmentName.String,
		FacilityName:    facility.FacilityName,
		Labels:          labels,
		Sections:        subSections,
		StartDate:       formatNullDate(report.WeekStart),
		StopDate:        formatNullDate(report.WeekStop),
		ReportID:        report.ReportID,
		Values:          clinicianEntryValuesFromReport(report),
		IsEdit:          true,
		StatusLabel:     reportSubmissionStatusLabel(report.SubmitStatus, report.ReportStatus),
		ReturnURL:       sanitizeReportSubmissionURL(c.Query("return_to")),
		ReadOnly:        true,
		WeekDays:        buildWeekDayChecks(report.WeekStart.Time, report.WeekStop.Time, report.DaysWorked.String),
		HideCoreSection: subHideCore,
		AbsenceReason:   report.AbsenceReason.String,
	}
	applyDynamicReportValues(c.Request.Context(), db, report.ReportID, resolveClinicianEntryKeys(c.Request.Context(), db, report.DepartmentID), entryForm.Values)
	sessionData.Form = entryForm

	utilities.GenerateHTML(c, sessionData, "base", "clinician-entry")
}

func HandlerReportSubmissionApprove(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {
	sessionData, ok := Get_Session_Data(c, db, sessionManager, nil).(utilities.TemplateData)
	if !ok {
		log.Println("Failed to retrieve session data as TemplateData")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to retrieve session data"})
		return
	}

	sesDetails, ok := sessionData.Ses.(utilities.SessionDetails)
	if !ok {
		log.Println("Failed to retrieve session details from TemplateData")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to retrieve session details"})
		return
	}

	reportID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid report ID")
		return
	}

	updated, err := models.ApproveFacilityReport(c.Request.Context(), db, reportID, sesDetails.HFID, sesDetails.EmpID)
	if err != nil {
		log.Printf("Error approving report submission: %v", err)
		c.String(http.StatusInternalServerError, "Unable to approve report submission")
		return
	}
	if !updated {
		c.String(http.StatusForbidden, "Only pending submitted reports within your facility can be approved")
		return
	}

	c.Redirect(http.StatusFound, sanitizeReportSubmissionURL(c.PostForm("return_to")))
}

func HandlerReportSubmissionSubmitAll(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {
	sessionData, ok := Get_Session_Data(c, db, sessionManager, nil).(utilities.TemplateData)
	if !ok {
		log.Println("Failed to retrieve session data as TemplateData")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to retrieve session data"})
		return
	}

	sesDetails, ok := sessionData.Ses.(utilities.SessionDetails)
	if !ok {
		log.Println("Failed to retrieve session details from TemplateData")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to retrieve session details"})
		return
	}
	if roleFromRights(sesDetails.Rights) != utilities.RoleFacilityAdmin {
		c.String(http.StatusForbidden, "Only facility admins can submit all reports for a week")
		return
	}

	department, _ := strconv.Atoi(c.PostForm("department"))
	year, _ := strconv.Atoi(c.PostForm("year"))
	week, _ := strconv.Atoi(c.PostForm("week"))
	if year <= 0 || week <= 0 {
		c.String(http.StatusBadRequest, "Select a specific year and week before submitting all reports")
		return
	}

	weekStart := isoWeekStart(year, week)
	weekStop := weekStart.AddDate(0, 0, 6)

	missingOnDuty, err := models.CountOnDutyEmployeesMissingWeeklyReport(c.Request.Context(), db, sesDetails.HFID, weekStart, weekStop, department)
	if err != nil {
		log.Printf("Error validating missing on-duty reports: %v", err)
		c.String(http.StatusInternalServerError, "Unable to validate on-duty report coverage")
		return
	}
	if missingOnDuty > 0 {
		c.String(http.StatusForbidden, "Some on-duty staff still have no report for this week. Complete all on-duty entries first.")
		return
	}

	if _, err := models.EnsureOnLeaveZeroReports(c.Request.Context(), db, sesDetails.HFID, weekStart, weekStop, sesDetails.EmpID, department); err != nil {
		log.Printf("Error creating on-leave zero reports: %v", err)
		c.String(http.StatusInternalServerError, "Unable to prepare on-leave records")
		return
	}

	if _, err := models.SubmitFacilityReportsByFilter(c.Request.Context(), db, sesDetails.HFID, 0, year, 0, week, sesDetails.EmpID); err != nil {
		if errors.Is(err, models.ErrFacilityBatchRequiresApprovedReports) {
			c.String(http.StatusForbidden, "Approve all reports for this facility week before submitting the weekly batch to national review")
			return
		}
		log.Printf("Error submitting facility reports: %v", err)
		c.String(http.StatusInternalServerError, "Unable to submit facility reports")
		return
	}

	c.Redirect(http.StatusFound, sanitizeReportSubmissionURL(c.PostForm("return_to")))
}

// HandlerReportSubmissionSubmitDay escalates all locally-approved reports for
// a specific calendar date to national review. Called via AJAX from the daily
// breakdown table. Accepts JSON {date:"YYYY-MM-DD", department:N}.
func HandlerReportSubmissionSubmitDay(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {
	sessionData, ok := Get_Session_Data(c, db, sessionManager, nil).(utilities.TemplateData)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": "session error"})
		return
	}
	sesDetails, ok := sessionData.Ses.(utilities.SessionDetails)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": "session error"})
		return
	}
	if roleFromRights(sesDetails.Rights) != utilities.RoleFacilityAdmin {
		c.JSON(http.StatusForbidden, gin.H{"ok": false, "error": "Only facility admins can submit daily reports to national"})
		return
	}

	var body struct {
		Date       string `json:"date"`
		Department int    `json:"department"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "invalid request body"})
		return
	}
	date, err := time.Parse("2006-01-02", body.Date)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "invalid date format (expected YYYY-MM-DD)"})
		return
	}

	count, err := models.SubmitFacilityReportsByDay(c.Request.Context(), db, sesDetails.HFID, body.Department, date, sesDetails.EmpID)
	if err != nil {
		if errors.Is(err, models.ErrFacilityBatchRequiresApprovedReports) {
			c.JSON(http.StatusForbidden, gin.H{"ok": false, "error": "All reports for this day must be locally approved before submitting to national review."})
			return
		}
		log.Printf("SubmitFacilityReportsByDay error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": "Unable to submit daily reports"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true, "count": count})
}

func HandlerAdminFacilitySubmissionApprove(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {
	handleAdminFacilitySubmissionDecision(c, db, sessionManager, true)
}

func HandlerAdminFacilitySubmissionDecline(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {
	handleAdminFacilitySubmissionDecision(c, db, sessionManager, false)
}

func handleAdminFacilitySubmissionDecision(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager, approve bool) {
	sessionData, ok := Get_Session_Data(c, db, sessionManager, nil).(utilities.TemplateData)
	if !ok {
		log.Println("Failed to retrieve session data as TemplateData")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to retrieve session data"})
		return
	}

	sesDetails, ok := sessionData.Ses.(utilities.SessionDetails)
	if !ok {
		log.Println("Failed to retrieve session details from TemplateData")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to retrieve session details"})
		return
	}
	if roleFromRights(sesDetails.Rights) != utilities.RoleNationalAdmin {
		c.String(http.StatusForbidden, "Only national admins can approve or decline by facility")
		return
	}

	facilityID, err := strconv.Atoi(c.Param("facility"))
	if err != nil || facilityID <= 0 {
		c.String(http.StatusBadRequest, "Invalid facility")
		return
	}

	department, _ := strconv.Atoi(c.PostForm("department"))
	year, _ := strconv.Atoi(c.PostForm("year"))
	month, _ := strconv.Atoi(c.PostForm("month"))
	week, _ := strconv.Atoi(c.PostForm("week"))

	if approve {
		_, err = models.ApproveNationalFacilityReportsByFilter(c.Request.Context(), db, int64(facilityID), department, year, month, week, sesDetails.EmpID)
	} else {
		_, err = models.DeclineNationalFacilityReportsByFilter(c.Request.Context(), db, int64(facilityID), department, year, month, week, sesDetails.EmpID)
	}
	if err != nil {
		log.Printf("Error processing facility-level review: %v", err)
		c.String(http.StatusInternalServerError, "Unable to process facility-level review")
		return
	}

	c.Redirect(http.StatusFound, sanitizeReportSubmissionURL(c.PostForm("return_to")))
}

func HandlerReportSubmissionApproveAll(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {
	sessionData, ok := Get_Session_Data(c, db, sessionManager, nil).(utilities.TemplateData)
	if !ok {
		log.Println("Failed to retrieve session data as TemplateData")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to retrieve session data"})
		return
	}

	sesDetails, ok := sessionData.Ses.(utilities.SessionDetails)
	if !ok {
		log.Println("Failed to retrieve session details from TemplateData")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to retrieve session details"})
		return
	}

	department, _ := strconv.Atoi(c.PostForm("department"))
	year, _ := strconv.Atoi(c.PostForm("year"))
	month, _ := strconv.Atoi(c.PostForm("month"))
	week, _ := strconv.Atoi(c.PostForm("week"))

	_, err := models.ApproveFacilityReportsByFilter(c.Request.Context(), db, sesDetails.HFID, department, year, month, week, sesDetails.EmpID)
	if err != nil {
		log.Printf("Error approving filtered report submissions: %v", err)
		c.String(http.StatusInternalServerError, "Unable to approve the selected report submissions")
		return
	}

	c.Redirect(http.StatusFound, sanitizeReportSubmissionURL(c.PostForm("return_to")))
}

func HandlerReportSubmissionDecline(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {
	sessionData, ok := Get_Session_Data(c, db, sessionManager, nil).(utilities.TemplateData)
	if !ok {
		log.Println("Failed to retrieve session data as TemplateData")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to retrieve session data"})
		return
	}

	sesDetails, ok := sessionData.Ses.(utilities.SessionDetails)
	if !ok {
		log.Println("Failed to retrieve session details from TemplateData")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to retrieve session details"})
		return
	}

	reportID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid report ID")
		return
	}

	updated, err := models.DeclineFacilityReport(c.Request.Context(), db, reportID, sesDetails.HFID, sesDetails.EmpID)
	if err != nil {
		log.Printf("Error declining report submission: %v", err)
		c.String(http.StatusInternalServerError, "Unable to decline report submission")
		return
	}
	if !updated {
		c.String(http.StatusForbidden, "Only pending submitted reports within your facility can be declined")
		return
	}

	c.Redirect(http.StatusFound, sanitizeReportSubmissionURL(c.PostForm("return_to")))
}

func buildReportSubmissionsURL(facilityID int, departmentID int, year int, month int, week int, status string) string {
	if year < 0 {
		year = 0
	}
	if month < 0 {
		month = 0
	}
	if week < 0 {
		week = 0
	}

	params := []string{}
	if facilityID > 0 {
		params = append(params, fmt.Sprintf("facility=%d", facilityID))
	}
	if departmentID > 0 {
		params = append(params, fmt.Sprintf("department=%d", departmentID))
	}
	// Preserve explicit period scope (including 0 = all) so KPI links and list totals stay aligned.
	params = append(params, fmt.Sprintf("year=%d", year))
	params = append(params, fmt.Sprintf("month=%d", month))
	params = append(params, fmt.Sprintf("week=%d", week))
	if status != "" && status != "all" {
		params = append(params, "status="+status)
	}
	if len(params) == 0 {
		return "/reports/analysis"
	}
	return "/reports/analysis?" + strings.Join(params, "&")
}

func buildReportSubmissionsURLWithView(facilityID int, departmentID int, year int, month int, week int, status string, view string) string {
	base := buildReportSubmissionsURL(facilityID, departmentID, year, month, week, status)
	if view == "" || view == "staff" {
		return base
	}
	if strings.Contains(base, "?") {
		return base + "&view=" + view
	}
	return base + "?view=" + view
}

func buildReportSubmissionsURLWithViewAndPage(facilityID int, departmentID int, year int, month int, week int, status string, view string, page int) string {
	base := buildReportSubmissionsURLWithView(facilityID, departmentID, year, month, week, status, view)
	if page <= 1 {
		return base
	}
	if strings.Contains(base, "?") {
		return base + fmt.Sprintf("&page=%d", page)
	}
	return base + fmt.Sprintf("?page=%d", page)
}

func buildReportSubmissionsURLWithModeAndPage(facilityID int, departmentID int, year int, month int, week int, status string, view string, mode string, page int) string {
	base := buildReportSubmissionsURLWithViewAndPage(facilityID, departmentID, year, month, week, status, view, page)
	trimmed := strings.TrimSpace(strings.ToLower(mode))
	if trimmed == "" || trimmed == "weeks" {
		trimmed = ""
	}
	if trimmed == "" {
		return base
	}
	if strings.Contains(base, "?") {
		return base + "&mode=" + trimmed
	}
	return base + "?mode=" + trimmed
}

func buildFacilityWeekDrilldownURL(facilityID int, departmentID int, status string, row *models.FacilitySubmissionSummaryRow) string {
	if row == nil || !row.WeekStart.Valid {
		return buildReportSubmissionsURLWithModeAndPage(facilityID, departmentID, 0, 0, 0, status, "staff", "reports", 1)
	}
	year, week := row.WeekStart.Time.ISOWeek()
	month := int(row.WeekStart.Time.Month())
	return buildReportSubmissionsURLWithModeAndPage(facilityID, departmentID, year, month, week, status, "staff", "days", 1)
}

func normalizeReportSubmissionMode(role string, value string) string {
	trimmed := strings.ToLower(strings.TrimSpace(value))
	if role == utilities.RoleStaff {
		switch trimmed {
		case "reports":
			return "reports"
		default:
			return "weeks"
		}
	}
	if role != utilities.RoleFacilityAdmin && role != utilities.RoleNationalAdmin {
		return "reports"
	}
	switch trimmed {
	case "reports":
		return "reports"
	case "days":
		return "days"
	default:
		return "weeks"
	}
}

func buildReportSubmissionsExportURL(format string, facilityID int, departmentID int, year int, month int, week int, status string) string {
	if year < 0 {
		year = 0
	}
	if month < 0 {
		month = 0
	}
	if week < 0 {
		week = 0
	}

	params := []string{}
	if facilityID > 0 {
		params = append(params, fmt.Sprintf("facility=%d", facilityID))
	}
	if departmentID > 0 {
		params = append(params, fmt.Sprintf("department=%d", departmentID))
	}
	params = append(params, fmt.Sprintf("year=%d", year))
	params = append(params, fmt.Sprintf("month=%d", month))
	params = append(params, fmt.Sprintf("week=%d", week))
	if status != "" && status != "all" {
		params = append(params, "status="+status)
	}
	if format != "" {
		params = append(params, "format="+format)
	}
	if len(params) == 0 {
		return "/reports/analysis/export"
	}
	return "/reports/analysis/export?" + strings.Join(params, "&")
}

func buildReportSubmissionsExportURLWithView(format string, facilityID int, departmentID int, year int, month int, week int, status string, view string) string {
	base := buildReportSubmissionsExportURL(format, facilityID, departmentID, year, month, week, status)
	if view == "" || view == "staff" {
		return base
	}
	if strings.Contains(base, "?") {
		return base + "&view=" + view
	}
	return base + "?view=" + view
}

func buildReportSubmissionsExportFilename(facilityID int, departmentID int, year int, month int, week int, status string) string {
	name := "report-submissions"
	if facilityID > 0 {
		name += fmt.Sprintf("-facility-%d", facilityID)
	}
	if departmentID > 0 {
		name += fmt.Sprintf("-department-%d", departmentID)
	}
	if year > 0 {
		name += fmt.Sprintf("-%d", year)
	}
	if month > 0 {
		name += fmt.Sprintf("-month-%02d", month)
	}
	if week > 0 {
		name += fmt.Sprintf("-week-%02d", week)
	}
	if status != "" && status != "all" {
		name += "-" + status
	}
	return name
}

func sanitizeReportSubmissionURL(value string) string {
	if value == "" {
		return "/reports/analysis"
	}
	if strings.HasPrefix(value, "/reports/analysis") || strings.HasPrefix(value, "/reports/review") {
		return value
	}
	return "/reports/analysis"
}

func reportSubmissionStatusLabel(submitStatus sql.NullString, reportStatus sql.NullString) string {
	if reportStatus.Valid {
		switch reportStatus.String {
		case "Approved":
			return "Approved"
		case "Rejected", "Declined":
			return "Declined"
		}
	}
	if submitStatus.Valid && submitStatus.String == "Submitted" {
		return "Submitted"
	}
	return "Draft"
}

func parseOptionalIntQuery(c *gin.Context, key string) (int, bool) {
	value, ok := c.GetQuery(key)
	if !ok {
		return 0, false
	}
	parsed, _ := strconv.Atoi(value)
	return parsed, true
}

func resolveReportSubmissionPeriod(c *gin.Context, db *sql.DB, requestedYear int, hasYear bool, requestedMonth int, hasMonth bool, requestedWeek int, hasWeek bool) (int, int, int, []int, []models.DashboardFilterOption, []models.ClinicianWeekOption, string, error) {
	periods, err := models.GetDashboardReportPeriods(c.Request.Context(), db)
	if err != nil {
		return 0, 0, 0, nil, nil, nil, "", err
	}
	if len(periods) == 0 {
		periods = []time.Time{startOfWeekLocal(time.Now())}
	}

	latestStart := periods[0]
	latestYear, latestWeek := latestStart.ISOWeek()
	latestMonth := int(latestStart.Month())

	// Default to "All records" on first load so the list reflects the entire
	// dataset. Once any period query param is supplied, fall back to the
	// latest-week defaults so partial filter combinations still resolve.
	anyPeriodRequested := hasYear || hasMonth || hasWeek
	selectedYear := 0
	selectedMonth := 0
	selectedWeek := 0
	if anyPeriodRequested {
		selectedYear = latestYear
		selectedMonth = latestMonth
		selectedWeek = latestWeek
	}
	if hasYear {
		selectedYear = requestedYear
	}
	if hasMonth {
		selectedMonth = requestedMonth
	}
	if hasWeek {
		selectedWeek = requestedWeek
	}

	availableYears := []int{0}
	yearSeen := map[int]bool{}
	for _, period := range periods {
		year, _ := period.ISOWeek()
		if yearSeen[year] {
			continue
		}
		yearSeen[year] = true
		availableYears = append(availableYears, year)
	}
	sort.SliceStable(availableYears[1:], func(i, j int) bool {
		return availableYears[i+1] > availableYears[j+1]
	})

	availableMonths := []models.DashboardFilterOption{{ID: 0, Name: "All"}}
	monthSeen := map[int]bool{}
	for _, period := range periods {
		year, _ := period.ISOWeek()
		if selectedYear > 0 && year != selectedYear {
			continue
		}
		month := int(period.Month())
		if monthSeen[month] {
			continue
		}
		monthSeen[month] = true
		availableMonths = append(availableMonths, models.DashboardFilterOption{
			ID:   month,
			Name: time.Month(month).String(),
		})
	}
	sort.SliceStable(availableMonths[1:], func(i, j int) bool {
		return availableMonths[i+1].ID > availableMonths[j+1].ID
	})

	availableWeeks := []models.ClinicianWeekOption{{
		Year:  0,
		Week:  0,
		Label: "All",
	}}
	weekSeen := map[string]bool{}
	for _, period := range periods {
		year, week := period.ISOWeek()
		month := int(period.Month())
		if selectedYear > 0 && year != selectedYear {
			continue
		}
		if selectedMonth > 0 && month != selectedMonth {
			continue
		}
		key := fmt.Sprintf("%d-%d-%02d", year, month, week)
		if weekSeen[key] {
			continue
		}
		weekSeen[key] = true
		availableWeeks = append(availableWeeks, models.ClinicianWeekOption{
			Year:      year,
			Week:      week,
			StartDate: period.Format("2006-01-02"),
			EndDate:   period.AddDate(0, 0, 6).Format("2006-01-02"),
			Label:     fmt.Sprintf("Week %02d (%s - %s)", week, period.Format("02 Jan"), period.AddDate(0, 0, 6).Format("02 Jan")),
		})
	}
	sort.SliceStable(availableWeeks[1:], func(i, j int) bool {
		return availableWeeks[i+1].StartDate > availableWeeks[j+1].StartDate
	})

	selectedWeekLabel := "Across all records"
	switch {
	case selectedYear > 0 && selectedMonth > 0 && selectedWeek > 0:
		selectedWeekLabel = fmt.Sprintf("Week %02d in %s %d", selectedWeek, time.Month(selectedMonth).String(), selectedYear)
	case selectedYear > 0 && selectedMonth > 0:
		selectedWeekLabel = fmt.Sprintf("All weeks in %s %d", time.Month(selectedMonth).String(), selectedYear)
	case selectedYear > 0 && selectedWeek > 0:
		selectedWeekLabel = fmt.Sprintf("Week %02d across %d", selectedWeek, selectedYear)
	case selectedMonth > 0 && selectedWeek > 0:
		selectedWeekLabel = fmt.Sprintf("Week %02d across %s", selectedWeek, time.Month(selectedMonth).String())
	case selectedMonth > 0:
		selectedWeekLabel = fmt.Sprintf("All years in %s", time.Month(selectedMonth).String())
	case selectedYear > 0:
		selectedWeekLabel = fmt.Sprintf("All months in %d", selectedYear)
	}

	return selectedYear, selectedMonth, selectedWeek, availableYears, availableMonths, availableWeeks, selectedWeekLabel, nil
}

func containsFilterOption(options []models.DashboardFilterOption, target int) bool {
	if target == 0 {
		return true
	}
	for _, option := range options {
		if option.ID == target {
			return true
		}
	}
	return false
}

func normalizeReportSubmissionStatus(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "submitted":
		return "submitted"
	case "pending":
		return "pending"
	case "approved":
		return "approved"
	case "declined":
		return "declined"
	case "draft":
		return "draft"
	default:
		return "all"
	}
}

func normalizeReportSubmissionViewMode(role string, value string) string {
	trimmed := strings.ToLower(strings.TrimSpace(value))
	if role != utilities.RoleNationalAdmin {
		return "staff"
	}
	if trimmed == "staff" {
		return "staff"
	}
	return "facility"
}

func isoWeekStart(year int, week int) time.Time {
	base := time.Date(year, time.January, 4, 0, 0, 0, 0, time.Local)
	weekday := int(base.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	week1Monday := base.AddDate(0, 0, -(weekday - 1))
	return week1Monday.AddDate(0, 0, (week-1)*7)
}

func reportSubmissionFilterSummary(role string, status string, periodLabel string) string {
	statusLabel := "all"
	switch status {
	case "submitted":
		statusLabel = "submitted"
	case "pending":
		statusLabel = "pending review"
	case "approved":
		statusLabel = "approved"
	case "declined":
		statusLabel = "declined"
	case "draft":
		statusLabel = "draft"
	}

	if role == utilities.RoleStaff {
		return fmt.Sprintf("Showing %s reports for %s.", statusLabel, periodLabel)
	}
	return fmt.Sprintf("Showing %s submissions for %s.", statusLabel, periodLabel)
}

func isFinalReportReviewStatus(status string) bool {
	switch status {
	case "Approved", "Rejected", "Declined":
		return true
	default:
		return false
	}
}

// HandlerReportsAnalysisWeekDays returns a JSON list of per-day submission
// stats for a given facility, ISO year, and week number. Used by the weekly
// accordion AJAX call to populate the inline day-breakdown table.
func HandlerReportsAnalysisWeekDays(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {
	sessionData, ok := Get_Session_Data(c, db, sessionManager, nil).(utilities.TemplateData)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "session error"})
		return
	}
	sesDetails, ok := sessionData.Ses.(utilities.SessionDetails)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "session error"})
		return
	}
	role := roleFromRights(sesDetails.Rights)
	if role != utilities.RoleFacilityAdmin && role != utilities.RoleNationalAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}

	facilityID, hasFacility := parseOptionalIntQuery(c, "facility")
	if !hasFacility || facilityID <= 0 {
		facilityID = int(sesDetails.HFID)
	}
	if role == utilities.RoleFacilityAdmin && facilityID != int(sesDetails.HFID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}

	departmentID, _ := parseOptionalIntQuery(c, "department")
	year, hasYear := parseOptionalIntQuery(c, "year")
	week, hasWeek := parseOptionalIntQuery(c, "week")
	if !hasYear || !hasWeek || year <= 0 || week <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "year and week are required"})
		return
	}

	weekStart := isoWeekStart(year, week)
	rows, err := models.GetWeekDailyBreakdown(c.Request.Context(), db, facilityID, departmentID, weekStart)
	if err != nil {
		log.Printf("GetWeekDailyBreakdown error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load week days"})
		return
	}

	type dayJSON struct {
		Date         string `json:"date"`
		DayName      string `json:"day_name"`
		Label        string `json:"label"`
		OnDutyCount  int    `json:"on_duty"`
		OnLeaveCount int    `json:"on_leave"`
		Submitted    int    `json:"submitted"`
		Approved     int    `json:"approved"`
		NotSubmitted int    `json:"not_submitted"`
		Percentage   int    `json:"pct"`
	}
	days := make([]dayJSON, 0, len(rows))
	for _, r := range rows {
		days = append(days, dayJSON{
			Date:         r.Day.Format("2006-01-02"),
			DayName:      r.DayName,
			Label:        r.DayName + ", " + r.Day.Format("02 Jan 2006"),
			OnDutyCount:  r.OnDutyCount,
			OnLeaveCount: r.OnLeaveCount,
			Submitted:    r.Submitted,
			Approved:     r.Approved,
			NotSubmitted: r.NotSubmitted,
			Percentage:   r.Percentage,
		})
	}
	c.JSON(http.StatusOK, gin.H{"days": days, "facility": facilityID, "department": departmentID})
}

// HandlerReportsAnalysisDayStaff returns a JSON list of all staff at the
// facility with their submission status for a given date. Used by the
// day-breakdown panel's AJAX call.
func HandlerReportsAnalysisDayStaff(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {
	sessionData, ok := Get_Session_Data(c, db, sessionManager, nil).(utilities.TemplateData)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "session error"})
		return
	}
	sesDetails, ok := sessionData.Ses.(utilities.SessionDetails)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "session error"})
		return
	}
	role := roleFromRights(sesDetails.Rights)
	if role != utilities.RoleFacilityAdmin && role != utilities.RoleNationalAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}

	facilityID, hasFacility := parseOptionalIntQuery(c, "facility")
	if !hasFacility || facilityID <= 0 {
		facilityID = int(sesDetails.HFID)
	}
	if role == utilities.RoleFacilityAdmin && facilityID != int(sesDetails.HFID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}

	departmentID, _ := parseOptionalIntQuery(c, "department")

	dateStr := c.Query("date")
	if dateStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "date required"})
		return
	}
	date, err := time.ParseInLocation("2006-01-02", dateStr, time.Local)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid date format, expected YYYY-MM-DD"})
		return
	}

	staff, err := models.GetDayStaffStatus(c.Request.Context(), db, facilityID, departmentID, date)
	if err != nil {
		log.Printf("GetDayStaffStatus error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load staff"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"date":  dateStr,
		"staff": staff,
	})
}

// HandlerReportsAnalysisDayStaffEntry returns JSON with the labeled field values
// from a single staff member's report for a given day. Used by the expandable
// row in the day-staff pane.
func HandlerReportsAnalysisDayStaffEntry(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {
	sessionData, ok := Get_Session_Data(c, db, sessionManager, nil).(utilities.TemplateData)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "session error"})
		return
	}
	sesDetails, ok := sessionData.Ses.(utilities.SessionDetails)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "session error"})
		return
	}
	role := roleFromRights(sesDetails.Rights)
	if role != utilities.RoleFacilityAdmin && role != utilities.RoleNationalAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}

	reportID, hasID := parseOptionalIntQuery(c, "report_id")

	var scopeFacilityID int64
	if role == utilities.RoleFacilityAdmin {
		scopeFacilityID = sesDetails.HFID
	}

	type Field struct {
		Key        string `json:"key"`
		Label      string `json:"label"`
		Value      string `json:"value"`
		Editable   bool   `json:"editable"`
		Attendance bool   `json:"attendance,omitempty"`
	}

	labels := resolveClinicianEntryLabels(c.Request.Context(), db)

	// Schema-only mode: no report yet. Return the department's field schema
	// with empty values so the UI can render an editable inline form for the
	// facility admin to enter data on behalf of the staff member.
	if !hasID || reportID <= 0 {
		empIDStr := strings.TrimSpace(c.Query("employee"))
		dateStr := strings.TrimSpace(c.Query("date"))
		if empIDStr == "" || dateStr == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "report_id or (employee and date) required"})
			return
		}
		empID64, err := strconv.ParseInt(empIDStr, 10, 64)
		if err != nil || empID64 <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid employee"})
			return
		}
		employee, err := models.EmployeeByID(c.Request.Context(), db, int(empID64))
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "employee not found"})
			return
		}
		if role == utilities.RoleFacilityAdmin && employee.EmpFacility != sesDetails.HFID {
			c.JSON(http.StatusForbidden, gin.H{"error": "employee outside your facility"})
			return
		}
		keys := resolveClinicianEntryDisplayKeys(c.Request.Context(), db, employee.EmpDepartment)
		fields := make([]Field, 0, len(keys))
		for _, key := range keys {
			label := labels[key]
			if label == "" {
				label = key
			}
			fields = append(fields, Field{
				Key: key, Label: label, Value: "", Editable: true,
				Attendance: key == "attendance",
			})
		}
		c.JSON(http.StatusOK, gin.H{
			"report_id":     0,
			"has_report":    false,
			"employee_id":   empID64,
			"date":          dateStr,
			"submit_status": "",
			"report_status": "",
			"absence_reason": "",
			"fields":        fields,
		})
		return
	}

	report, err := models.GetReportSubmissionByIDForReview(c.Request.Context(), db, reportID, scopeFacilityID, role)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "report not found"})
		return
	}

	values := clinicianEntryValuesFromReport(report)

	fieldOrder := []string{
		"attendance", "ward_rounds", "patients_reviewed", "OPD_clinics", "OPD_patients",
		"anc_patients", "theatre_days", "elective", "emergency", "postmortems",
		"teaching_rounds", "students_taught", "mortality_reviews",
		"maternal", "perinatal", "surgical", "medical", "paed",
		"labs_requests", "imaging_requests", "lab_investigations",
		"BS", "HIV", "malaria", "TB", "CBC", "chemistry", "hematology", "urinalysis",
		"gram_stain", "culture", "microbiology", "sensitivity_tests",
		"diagnostics", "xrays", "ct_scans", "obstetrics_scans", "abdominal_scans",
	}

	fields := []Field{}
	for _, key := range fieldOrder {
		val := values[key]
		if val == "" || val == "0" {
			continue
		}
		label := labels[key]
		if label == "" {
			label = key
		}
		fields = append(fields, Field{Key: key, Label: label, Value: val})
	}

	submitStatus := ""
	if report.SubmitStatus.Valid {
		submitStatus = report.SubmitStatus.String
	}
	reportStatus := ""
	if report.ReportStatus.Valid {
		reportStatus = report.ReportStatus.String
	}

	c.JSON(http.StatusOK, gin.H{
		"report_id":      reportID,
		"has_report":     true,
		"submit_status":  submitStatus,
		"report_status":  reportStatus,
		"absence_reason": report.AbsenceReason.String,
		"fields":         fields,
	})
}

// HandlerReportsAnalysisDayStaffReview handles approve/decline actions from the
// day-staff accordion pane. Expects JSON: {"report_id": N, "action": "approve"|"decline"}.
// Only Facility Admins may call this; they can only act on reports within their own facility.
func HandlerReportsAnalysisDayStaffReview(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {
	sessionData, ok := Get_Session_Data(c, db, sessionManager, nil).(utilities.TemplateData)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "session error"})
		return
	}
	sesDetails, ok := sessionData.Ses.(utilities.SessionDetails)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "session error"})
		return
	}
	role := roleFromRights(sesDetails.Rights)
	if role != utilities.RoleFacilityAdmin && role != utilities.RoleNationalAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}

	var body struct {
		ReportID int    `json:"report_id"`
		Action   string `json:"action"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.ReportID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "report_id and action required"})
		return
	}

	facilityID := sesDetails.HFID
	approverID := sesDetails.EmpID

	var (
		updated   bool
		err       error
		newStatus string
	)
	switch strings.ToLower(body.Action) {
	case "approve":
		updated, err = models.ApproveFacilityReportDirect(c.Request.Context(), db, body.ReportID, facilityID, approverID)
		newStatus = "Approved"
	case "decline":
		updated, err = models.DeclineFacilityReportDirect(c.Request.Context(), db, body.ReportID, facilityID, approverID)
		newStatus = "Declined"
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "action must be 'approve' or 'decline'"})
		return
	}

	if err != nil {
		log.Printf("HandlerReportsAnalysisDayStaffReview: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update report"})
		return
	}
	if !updated {
		c.JSON(http.StatusConflict, gin.H{"error": "report could not be updated — it may already be reviewed or not submitted"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true, "status": newStatus})
}

// HandlerReportsAnalysisDayApproveAll approves every non-reviewed report for a
// specific day within the facility admin's own facility.
// Expects JSON: {"date": "2006-01-02", "department": 0}
func HandlerReportsAnalysisDayApproveAll(c *gin.Context, db *sql.DB, sessionManager *scs.SessionManager) {
	sessionData, ok := Get_Session_Data(c, db, sessionManager, nil).(utilities.TemplateData)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "session error"})
		return
	}
	sesDetails, ok := sessionData.Ses.(utilities.SessionDetails)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "session error"})
		return
	}
	role := roleFromRights(sesDetails.Rights)
	if role != utilities.RoleFacilityAdmin && role != utilities.RoleNationalAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}

	var body struct {
		Date         string `json:"date"`
		DepartmentID int    `json:"department"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.Date == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "date required"})
		return
	}

	date, err := time.Parse("2006-01-02", body.Date)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid date format"})
		return
	}

	count, err := models.ApproveFacilityReportsByDate(c.Request.Context(), db, sesDetails.HFID, body.DepartmentID, date, sesDetails.EmpID)
	if err != nil {
		log.Printf("HandlerReportsAnalysisDayApproveAll: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to approve reports"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true, "count": count})
}
