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
	router.POST("/forgot", func(c *gin.Context) { handlers.LoginHandler(c, db, sessionManager) })

	router.GET("/login", func(c *gin.Context) { handlers.HandlerLoginForm(c, db, sessionManager) })

	router.GET("/about", handlers.AboutHandler)
	router.GET("/forget", handlers.HandlerLoginForgot)
	router.GET("/help", handlers.HandlerHelp)

	// Protected routes
	protected := router.Group("/")
	protected.Use(middleware.AuthMiddleware(sessionManager)) // Apply the authentication middleware
	{
		// Home route
		protected.GET("/", func(c *gin.Context) { handlers.HandlerHome(c, db, sessionManager) })
		protected.GET("/logout", func(c *gin.Context) { handlers.LogoutHandler(c, sessionManager) })

		// additional routes
		RouteReports(protected, db, sessionManager)
		RouteFacilities(protected, db, sessionManager)
		RouteDepartment(protected, db, sessionManager)
		RouteEmployeee(protected, db, sessionManager)
	}

}

func RouteReports(r *gin.RouterGroup, db *sql.DB, sessionManager *scs.SessionManager) {
	v := r.Group("/reports")
	{
		//v.GET("/singleentry/:i", func(c *gin.Context) { handlers.SingleEntryList(c, db, sessionManager) })
		v.GET("/new/:i", func(c *gin.Context) { handlers.SingleEntryForm(c, db, sessionManager) })
		v.GET("/bulk", func(c *gin.Context) { handlers.HandlerBulkCaptureList(c, db, sessionManager) })
		v.GET("/entry", func(c *gin.Context) { handlers.HandlerBulkCaptureForm2(c, db, sessionManager) })
		v.GET("/export/:i", func(c *gin.Context) { handlers.HandlerReportExport(c, db, sessionManager) })
		v.GET("/view/:id", func(c *gin.Context) { handlers.HandlerReportView2(c, db, sessionManager) })
		v.GET("/view/json", func(c *gin.Context) { handlers.HandlerReportData2(c, db, sessionManager) })
		v.GET("/del/:i", func(c *gin.Context) { handlers.HandlerReportRem(c, db, sessionManager) })
		v.POST("/save", func(c *gin.Context) { handlers.HandlerReportSave(c, db, sessionManager) })
		v.POST("/zave", func(c *gin.Context) { handlers.HandlerReportZave(c, db, sessionManager) })
		v.POST("/update", func(c *gin.Context) { handlers.HandlerReportEntryUpdate(c, db, sessionManager) })
		v.POST("/filter", func(c *gin.Context) { handlers.HandlerReportList(c, db, sessionManager) })
		v.GET("/submissions", func(c *gin.Context) { handlers.HandlerReportFacilities2(c, db, sessionManager) })
		v.GET("/submissions/departments", func(c *gin.Context) { handlers.HandlerReportFacilities2(c, db, sessionManager) })
		v.GET("/list", func(c *gin.Context) { handlers.HandlerReportList2(c, db, sessionManager) })

		// New route for specific facility reports
		v.GET("/list/:facility", func(c *gin.Context) { handlers.HandlerReportList(c, db, sessionManager) })

		//Approve route to invoke Approve Handler.
		v.POST("/approve", func(c *gin.Context) { handlers.HandlerReportApprove(c, db, sessionManager) })
		v.POST("/submit", func(c *gin.Context) { handlers.HandlerReportApprove(c, db, sessionManager) })

		//Report summary and analysis routes
		v.GET("/analysis", func(c *gin.Context) { handlers.HandlerReportsAnalysis(c, db, sessionManager) })
		v.GET("/report-summaries", func(c *gin.Context) { handlers.HandlerReportSummaries(c, db, sessionManager) })

	}

}

func RouteFacilities(r *gin.RouterGroup, db *sql.DB, sessionManager *scs.SessionManager) {
	v := r.Group("/facilities")
	{
		v.GET("/new/:i", func(c *gin.Context) { handlers.HandlerFacilityForm(c, db, sessionManager) })
		v.POST("/save", func(c *gin.Context) { handlers.HandlerFacilitySave(c, db, sessionManager) })
		v.POST("/filter", func(c *gin.Context) { handlers.HandlerFacilityList(c, db, sessionManager) })
		v.GET("/list", func(c *gin.Context) { handlers.HandlerFacilityList(c, db, sessionManager) })
	}

}

func RouteDepartment(r *gin.RouterGroup, db *sql.DB, sessionManager *scs.SessionManager) {
	v := r.Group("/department")
	{
		v.GET("/new/:i", func(c *gin.Context) { handlers.HandlerDepartmentForm(c, db, sessionManager) })
		v.POST("/save", func(c *gin.Context) { handlers.HandlerDepartmentSave(c, db, sessionManager) })
		v.POST("/filter", func(c *gin.Context) { handlers.HandlerDepartmentList(c, db, sessionManager) })
		v.GET("/list", func(c *gin.Context) { handlers.HandlerDepartmentList(c, db, sessionManager) })
	}

}

func RouteEmployeee(r *gin.RouterGroup, db *sql.DB, sessionManager *scs.SessionManager) {
	v := r.Group("/employee")
	{
		v.GET("/new", func(c *gin.Context) { handlers.HandlerEmployeeForm(c, db, sessionManager) })
		v.POST("/save", func(c *gin.Context) { handlers.HandlerEmployeeSave(c, db, sessionManager) })
		v.POST("/filter", func(c *gin.Context) { handlers.HandlerEmployeeList(c, db, sessionManager) })
		v.GET("/list", func(c *gin.Context) { handlers.HandlerEmployeeList(c, db, sessionManager) })
		v.GET("/leave/form", func(c *gin.Context) { handlers.HandlerEmployeeLeaveForm(c, db, sessionManager) })
		v.POST("/leave/save", func(c *gin.Context) { handlers.HandlerEmployeeLeaveSave(c, db, sessionManager) })
		v.GET("/leave/staffonleave", func(c *gin.Context) { handlers.HandlerLeaveList(c, db, sessionManager) })
		v.GET("/leave/history", func(c *gin.Context) { handlers.HandlerEmployeeLeaveHistory(c, db, sessionManager) })
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
