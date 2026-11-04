package v1

import (
	"net/http"

	"hr-system/internal/handler/response"

	"github.com/gin-gonic/gin"
)

func (h *Handler) DepartmentReport(c *gin.Context) {
	rows, err := h.services.Report.DepartmentReport(c.Request.Context())
	if err != nil {
		response.ServiceError(c, err)
		return
	}
	result := make([]departmentReportResponse, 0, len(rows))
	for _, r := range rows {
		result = append(result, departmentReportResponse{
			DepartmentID:   r.DepartmentID,
			Department:     r.Department,
			EmployeesCount: r.EmployeesCount,
			TotalSalary:    r.TotalSalary,
			AvgSalary:      r.AvgSalary,
		})
	}
	response.JSON(c, http.StatusOK, result)
}
