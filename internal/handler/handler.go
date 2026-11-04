package handler

import (
	"hr-system/internal/handler/middleware"
	"hr-system/internal/handler/v1"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

type Handler struct {
	v1         *v1.Handler
	authn      middleware.Authenticator
	corsOrigin string
}

func NewHandler(v1h *v1.Handler, authn middleware.Authenticator, corsOrigin string) *Handler {
	return &Handler{v1: v1h, authn: authn, corsOrigin: corsOrigin}
}

func (h *Handler) InitRoutes() *gin.Engine {
	binding.EnableDecoderDisallowUnknownFields = true

	router := gin.New()

	router.Use(
		middleware.RequestID(),
		middleware.CORS(h.corsOrigin),
		middleware.Logging(),
		middleware.Recover(),
	)

	requireAuth := middleware.Auth(h.authn)
	requireHR := middleware.RequireRole("hr")

	router.GET("/health", h.v1.Health)

	api := router.Group("/api/v1")

	api.POST("/auth/login", h.v1.Login)
	api.POST("/auth/refresh", h.v1.Refresh)
	api.POST("/auth/logout", h.v1.Logout)

	authed := api.Group("", requireAuth)
	authed.GET("/me", h.v1.Me)
	authed.GET("/me/salaries", h.v1.MySalaries)
	authed.POST("/time-off", h.v1.CreateTimeOff)
	authed.GET("/time-off", h.v1.ListTimeOff)

	hr := api.Group("", requireAuth, requireHR)

	hr.POST("/departments", h.v1.CreateDepartment)
	hr.GET("/departments", h.v1.ListDepartments)
	hr.GET("/departments/:id", h.v1.GetDepartment)
	hr.PUT("/departments/:id", h.v1.UpdateDepartment)
	hr.DELETE("/departments/:id", h.v1.DeleteDepartment)

	hr.POST("/positions", h.v1.CreatePosition)
	hr.GET("/positions", h.v1.ListPositions)
	hr.GET("/positions/:id", h.v1.GetPosition)
	hr.PUT("/positions/:id", h.v1.UpdatePosition)
	hr.DELETE("/positions/:id", h.v1.DeletePosition)

	hr.POST("/employees", h.v1.CreateEmployee)
	hr.GET("/employees", h.v1.ListEmployees)
	hr.GET("/employees/export.csv", h.v1.ExportEmployees)
	hr.POST("/employees/import", h.v1.ImportEmployees)
	hr.GET("/employees/:id", h.v1.GetEmployee)
	hr.PUT("/employees/:id", h.v1.UpdateEmployee)
	hr.DELETE("/employees/:id", h.v1.DeleteEmployee)

	hr.POST("/employees/:id/salaries", h.v1.AddSalary)
	hr.GET("/employees/:id/salaries", h.v1.ListEmployeeSalaries)
	hr.GET("/employees/:id/history", h.v1.EmployeeHistory)

	hr.PATCH("/time-off/:id/status", h.v1.DecideTimeOff)

	hr.GET("/reports/departments", h.v1.DepartmentReport)

	return router
}
