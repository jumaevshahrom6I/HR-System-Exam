package v1

import (
	"bytes"
	"net/http"

	"hr-system/internal/handler/response"

	"github.com/gin-gonic/gin"
)

const maxImportFileBytes = 1 << 20

func (h *Handler) ExportEmployees(c *gin.Context) {
	filter, ok := parseEmployeeFilter(c)
	if !ok {
		return
	}

	var buf bytes.Buffer
	if err := h.services.Employee.Export(c.Request.Context(), filter, &buf); err != nil {
		response.ServiceError(c, err)
		return
	}

	c.Header("Content-Disposition", `attachment; filename="employees.csv"`)
	c.Data(http.StatusOK, "text/csv; charset=utf-8", buf.Bytes())
}

func (h *Handler) ImportEmployees(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxImportFileBytes+(64<<10))

	header, err := c.FormFile("file")
	if err != nil {
		response.Error(c, http.StatusBadRequest, "file is required (multipart field \"file\", max 1 MB)")
		return
	}
	if header.Size > maxImportFileBytes {
		response.Error(c, http.StatusBadRequest, "file is too large, max 1 MB")
		return
	}
	file, err := header.Open()
	if err != nil {
		response.Error(c, http.StatusBadRequest, "cannot read file")
		return
	}
	defer file.Close()

	result, err := h.services.Employee.Import(c.Request.Context(), file)
	if err != nil {
		response.ServiceError(c, err)
		return
	}
	response.JSON(c, http.StatusOK, newImportResponse(result))
}
