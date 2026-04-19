package routes

import (
	"database/sql"

	"github.com/alexedwards/scs/v2"
	"github.com/gin-gonic/gin"

	"github.com/moh/clinician/internals/handlers"
	"github.com/moh/clinician/internals/middleware"
)

func SetupRoutes(router *gin.Engine, db *sql.DB, sessionManager *scs.SessionManager) {

	// Public routes
	router.POST("/login", func(c *gin.Context) { handlers.LoginHandler(c, db, sessionManager) })
	router.POST("/setup", func(c *gin.Context) { handlers.SetupHandler(c, db) })
	router.POST("/register", func(c *gin.Context) { handlers.HandlerRegistrationSave(c, db, sessionManager) })
	router.POST("/forgot", func(c *gin.Context) { handlers.LoginHandler(c, db, sessionManager) })

	router.GET("/login", func(c *gin.Context) { handlers.HandlerLoginForm(c, db, sessionManager) })
	router.GET("/register", func(c *gin.Context) { handlers.HandlerRegistrationForm(c, db, sessionManager) })

	router.GET("/about", handlers.AboutHandler)
	router.GET("/forget", handlers.HandlerLoginForgot)
	router.GET("/help", handlers.HandlerHelp)

	// Protected routes
	protected := router.Group("/")
	protected.Use(middleware.AuthMiddleware(sessionManager)) // Apply the authentication middleware
	{
		// Home route
		protected.GET("/", func(c *gin.Context) { handlers.HandlerHome(c, db, sessionManager) })
		protected.GET("/account/edit", func(c *gin.Context) { handlers.HandlerAccountEdit(c, db, sessionManager) })
		protected.POST("/account/edit", func(c *gin.Context) { handlers.HandlerAccountUpdate(c, db, sessionManager) })
		protected.GET("/logout", func(c *gin.Context) { handlers.LogoutHandler(c, sessionManager) })

		// additional routes
		RouteReports(protected, db, sessionManager)
		RouteFacilities(protected, db, sessionManager)
		RouteDepartment(protected, db, sessionManager)
		RouteCustomization(protected, db, sessionManager)
		RouteEmployeee(protected, db, sessionManager)
		RouteLeave(protected, db, sessionManager)
	}

}

func RouteReports(r *gin.RouterGroup, db *sql.DB, sessionManager *scs.SessionManager) {
	v := r.Group("/reports")
	{
		v.GET("/new/:i", func(c *gin.Context) { handlers.SingleEntryForm(c, db, sessionManager) })
		v.GET("/analysis", func(c *gin.Context) { handlers.HandlerReportsAnalysis(c, db, sessionManager) })
		v.GET("/analysis/export", func(c *gin.Context) { handlers.HandlerReportsAnalysisExport(c, db, sessionManager) })
		v.GET("/history", func(c *gin.Context) { handlers.HandlerClinicianReportHistory(c, db, sessionManager) })
		v.GET("/history/export", func(c *gin.Context) { handlers.HandlerClinicianReportHistoryExport(c, db, sessionManager) })
		v.GET("/edit/:id", func(c *gin.Context) { handlers.HandlerClinicianReportEditForm(c, db, sessionManager) })
		v.POST("/zave", func(c *gin.Context) { handlers.HandlerReportZave(c, db, sessionManager) })
		v.POST("/update-self/:id", func(c *gin.Context) { handlers.HandlerClinicianReportUpdate(c, db, sessionManager) })
		v.POST("/delete-self/:id", func(c *gin.Context) { handlers.HandlerClinicianReportDelete(c, db, sessionManager) })
		v.POST("/submit-self/:id", func(c *gin.Context) { handlers.HandlerClinicianReportSubmit(c, db, sessionManager) })
	}

	managerOrAdmin := v.Group("/")
	managerOrAdmin.Use(middleware.RequireRoles(db, sessionManager, "admin", "approver"))
	{
		managerOrAdmin.GET("/entry", func(c *gin.Context) { handlers.HandlerBulkCaptureForm2(c, db, sessionManager) })
		managerOrAdmin.GET("/export/:i", func(c *gin.Context) { handlers.HandlerReportExport(c, db, sessionManager) })
		managerOrAdmin.GET("/view/:id", func(c *gin.Context) { handlers.HandlerReportView2(c, db, sessionManager) })
		managerOrAdmin.GET("/view/json", func(c *gin.Context) { handlers.HandlerReportData2(c, db, sessionManager) })
		managerOrAdmin.GET("/del/:i", func(c *gin.Context) { handlers.HandlerReportRem(c, db, sessionManager) })
		managerOrAdmin.POST("/save", func(c *gin.Context) { handlers.HandlerReportSave(c, db, sessionManager) })
		managerOrAdmin.POST("/update", func(c *gin.Context) { handlers.HandlerReportEntryUpdate(c, db, sessionManager) })
		managerOrAdmin.POST("/filter", func(c *gin.Context) { handlers.HandlerReportList(c, db, sessionManager) })
		managerOrAdmin.GET("/submissions", func(c *gin.Context) { handlers.HandlerReportFacilities2(c, db, sessionManager) })
		managerOrAdmin.GET("/submissions/departments", func(c *gin.Context) { handlers.HandlerReportFacilities2(c, db, sessionManager) })
		managerOrAdmin.GET("/list", func(c *gin.Context) { handlers.HandlerReportList2(c, db, sessionManager) })
		managerOrAdmin.GET("/list/:facility", func(c *gin.Context) { handlers.HandlerReportList(c, db, sessionManager) })
		managerOrAdmin.GET("/analysis/view/:id", func(c *gin.Context) { handlers.HandlerReportSubmissionView(c, db, sessionManager) })
		managerOrAdmin.POST("/analysis/submit-all", func(c *gin.Context) { handlers.HandlerReportSubmissionSubmitAll(c, db, sessionManager) })
		managerOrAdmin.POST("/analysis/facility/:facility/approve", func(c *gin.Context) { handlers.HandlerAdminFacilitySubmissionApprove(c, db, sessionManager) })
		managerOrAdmin.POST("/analysis/facility/:facility/decline", func(c *gin.Context) { handlers.HandlerAdminFacilitySubmissionDecline(c, db, sessionManager) })
		managerOrAdmin.GET("/report-summaries", func(c *gin.Context) { handlers.HandlerReportSummaries(c, db, sessionManager) })
	}

	approverOnly := v.Group("/")
	approverOnly.Use(middleware.RequireRoles(db, sessionManager, "approver"))
	{
		approverOnly.GET("/bulk", func(c *gin.Context) { handlers.HandlerBulkCaptureList(c, db, sessionManager) })
		approverOnly.POST("/approve", func(c *gin.Context) { handlers.HandlerReportApprove(c, db, sessionManager) })
		approverOnly.POST("/submit", func(c *gin.Context) { handlers.HandlerReportApprove(c, db, sessionManager) })
		approverOnly.GET("/review", func(c *gin.Context) { handlers.HandlerFacilityReportReview(c, db, sessionManager) })
		approverOnly.GET("/review/export", func(c *gin.Context) { handlers.HandlerFacilityReportReviewExport(c, db, sessionManager) })
		approverOnly.POST("/review/:id/approve", func(c *gin.Context) { handlers.HandlerFacilityReportApprove(c, db, sessionManager) })
		approverOnly.POST("/review/:id/decline", func(c *gin.Context) { handlers.HandlerFacilityReportDecline(c, db, sessionManager) })
		approverOnly.POST("/analysis/:id/approve", func(c *gin.Context) { handlers.HandlerReportSubmissionApprove(c, db, sessionManager) })
		approverOnly.POST("/analysis/:id/decline", func(c *gin.Context) { handlers.HandlerReportSubmissionDecline(c, db, sessionManager) })
		approverOnly.POST("/analysis/approve-all", func(c *gin.Context) { handlers.HandlerReportSubmissionApproveAll(c, db, sessionManager) })
	}

}

func RouteFacilities(r *gin.RouterGroup, db *sql.DB, sessionManager *scs.SessionManager) {
	v := r.Group("/facilities")
	v.Use(middleware.RequireRoles(db, sessionManager, "admin"))
	{
		v.GET("/new/:i", func(c *gin.Context) { handlers.HandlerFacilityForm(c, db, sessionManager) })
		v.POST("/save", func(c *gin.Context) { handlers.HandlerFacilitySave(c, db, sessionManager) })
		v.POST("/filter", func(c *gin.Context) { handlers.HandlerFacilityList(c, db, sessionManager) })
		v.GET("/list", func(c *gin.Context) { handlers.HandlerFacilityList(c, db, sessionManager) })
	}

}

func RouteDepartment(r *gin.RouterGroup, db *sql.DB, sessionManager *scs.SessionManager) {
	v := r.Group("/department")
	v.Use(middleware.RequireRoles(db, sessionManager, "admin"))
	{
		v.GET("/new/:i", func(c *gin.Context) { handlers.HandlerDepartmentForm(c, db, sessionManager) })
		v.POST("/save", func(c *gin.Context) { handlers.HandlerDepartmentSave(c, db, sessionManager) })
		v.POST("/filter", func(c *gin.Context) { handlers.HandlerDepartmentList(c, db, sessionManager) })
		v.GET("/list", func(c *gin.Context) { handlers.HandlerDepartmentList(c, db, sessionManager) })
	}

}

func RouteCustomization(r *gin.RouterGroup, db *sql.DB, sessionManager *scs.SessionManager) {
	v := r.Group("/customization")
	v.Use(middleware.RequireRoles(db, sessionManager, "admin"))
	{
		v.GET("", func(c *gin.Context) { handlers.HandlerCustomization(c, db, sessionManager) })
		v.GET("/", func(c *gin.Context) { handlers.HandlerCustomization(c, db, sessionManager) })
		v.POST("/facility/save", func(c *gin.Context) { handlers.HandlerCustomizationFacilitySave(c, db, sessionManager) })
		v.POST("/facility/delete/:id", func(c *gin.Context) { handlers.HandlerCustomizationFacilityDelete(c, db, sessionManager) })
		v.POST("/department/save", func(c *gin.Context) { handlers.HandlerCustomizationDepartmentSave(c, db, sessionManager) })
		v.POST("/department/delete/:id", func(c *gin.Context) { handlers.HandlerCustomizationDepartmentDelete(c, db, sessionManager) })
		v.POST("/role/save", func(c *gin.Context) { handlers.HandlerCustomizationRoleSave(c, db, sessionManager) })
		v.POST("/role/delete/:id", func(c *gin.Context) { handlers.HandlerCustomizationRoleDelete(c, db, sessionManager) })
		v.POST("/clinical-role/save", func(c *gin.Context) { handlers.HandlerCustomizationClinicalRoleSave(c, db, sessionManager) })
		v.POST("/clinical-role/delete/:id", func(c *gin.Context) { handlers.HandlerCustomizationClinicalRoleDelete(c, db, sessionManager) })
		v.POST("/data-element/save", func(c *gin.Context) { handlers.HandlerCustomizationDataElementSave(c, db, sessionManager) })
		v.POST("/data-element/delete/:id", func(c *gin.Context) { handlers.HandlerCustomizationDataElementDelete(c, db, sessionManager) })
	}
}

func RouteEmployeee(r *gin.RouterGroup, db *sql.DB, sessionManager *scs.SessionManager) {
	v := r.Group("/employee")
	v.Use(middleware.RequireRoles(db, sessionManager, "admin", "approver"))
	{
		v.GET("/new", func(c *gin.Context) { handlers.HandlerEmployeeForm(c, db, sessionManager) })
		v.GET("/edit/:id", func(c *gin.Context) { handlers.HandlerEmployeeEdit(c, db, sessionManager) })
		v.POST("/save", func(c *gin.Context) { handlers.HandlerEmployeeSave(c, db, sessionManager) })
		v.POST("/delete/:id", func(c *gin.Context) { handlers.HandlerEmployeeDelete(c, db, sessionManager) })
		v.POST("/filter", func(c *gin.Context) { handlers.HandlerEmployeeList(c, db, sessionManager) })
		v.GET("/list", func(c *gin.Context) { handlers.HandlerEmployeeList(c, db, sessionManager) })
		v.GET("/leave/form", func(c *gin.Context) { handlers.HandlerEmployeeLeaveForm(c, db, sessionManager) })
		v.POST("/leave/save", func(c *gin.Context) { handlers.HandlerEmployeeLeaveSave(c, db, sessionManager) })
		v.GET("/leave/staffonleave", func(c *gin.Context) { handlers.HandlerLeaveList(c, db, sessionManager) })
		v.GET("/:id/leave/history", func(c *gin.Context) { handlers.HandlerEmployeeLeaveHistory(c, db, sessionManager) })
		v.GET("/:id/leave/history/export", func(c *gin.Context) { handlers.HandlerEmployeeLeaveHistoryExport(c, db, sessionManager) })
	}

}

func RouteLeave(r *gin.RouterGroup, db *sql.DB, sessionManager *scs.SessionManager) {
	v := r.Group("/leave")
	userOnly := v.Group("/")
	userOnly.Use(middleware.RequireRoles(db, sessionManager, "user"))
	{
		userOnly.GET("/form", func(c *gin.Context) { handlers.HandlerEmployeeLeaveForm(c, db, sessionManager) })
		userOnly.POST("/save", func(c *gin.Context) { handlers.HandlerEmployeeLeaveSave(c, db, sessionManager) })
		userOnly.GET("/edit/:id", func(c *gin.Context) { handlers.HandlerEmployeeLeaveEditForm(c, db, sessionManager) })
		userOnly.POST("/update/:id", func(c *gin.Context) { handlers.HandlerEmployeeLeaveUpdate(c, db, sessionManager) })
		userOnly.POST("/delete/:id", func(c *gin.Context) { handlers.HandlerEmployeeLeaveDelete(c, db, sessionManager) })
		userOnly.GET("/history", func(c *gin.Context) { handlers.HandlerEmployeeLeaveHistory(c, db, sessionManager) })
		userOnly.GET("/history/export", func(c *gin.Context) { handlers.HandlerEmployeeLeaveHistoryExport(c, db, sessionManager) })
	}

	approverOnly := v.Group("/")
	approverOnly.Use(middleware.RequireRoles(db, sessionManager, "approver"))
	{
		approverOnly.GET("/review", func(c *gin.Context) { handlers.HandlerFacilityLeaveReview(c, db, sessionManager) })
		approverOnly.GET("/review/export", func(c *gin.Context) { handlers.HandlerFacilityLeaveReviewExport(c, db, sessionManager) })
		approverOnly.POST("/review/:id/approve", func(c *gin.Context) { handlers.HandlerFacilityLeaveApprove(c, db, sessionManager) })
		approverOnly.POST("/review/:id/decline", func(c *gin.Context) { handlers.HandlerFacilityLeaveDecline(c, db, sessionManager) })
	}
}

func RouteUnit(r *gin.RouterGroup, db *sql.DB, sessionManager *scs.SessionManager) {
	v := r.Group("/unit")
	{
		v.GET("/new/:i", func(c *gin.Context) { handlers.HandlerUnitForm(c, db, sessionManager) })
		v.POST("/save", func(c *gin.Context) { handlers.HandlerUnitSave(c, db, sessionManager) })
		v.POST("/filter", func(c *gin.Context) { handlers.HandlerUnitList(c, db, sessionManager) })
		v.GET("/list", func(c *gin.Context) { handlers.HandlerUnitList(c, db, sessionManager) })
	}

}

func RouteUser(r *gin.RouterGroup, db *sql.DB, sessionManager *scs.SessionManager) {
	v := r.Group("/user")
	{
		v.GET("/new/:i", func(c *gin.Context) { handlers.HandlerUserForm(c, db, sessionManager) })
		v.POST("/save", func(c *gin.Context) { handlers.HandlerUserSave(c, db, sessionManager) })
		v.POST("/filter", func(c *gin.Context) { handlers.HandlerUserList(c, db, sessionManager) })
		v.GET("/list", func(c *gin.Context) { handlers.HandlerUserList(c, db, sessionManager) })
	}

}

func RouteUserright(r *gin.RouterGroup, db *sql.DB, sessionManager *scs.SessionManager) {
	v := r.Group("/userright")
	{
		v.GET("/new/:i", func(c *gin.Context) { handlers.HandlerUserRightForm(c, db, sessionManager) })
		v.POST("/save", func(c *gin.Context) { handlers.HandlerUserRightSave(c, db, sessionManager) })
		v.POST("/filter", func(c *gin.Context) { handlers.HandlerUserRightList(c, db, sessionManager) })
		v.GET("/list", func(c *gin.Context) { handlers.HandlerUserRightList(c, db, sessionManager) })
	}

}
