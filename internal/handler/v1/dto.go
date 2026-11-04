package v1

import (
	"time"

	"hr-system/internal/models"
)

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

func newTokenResponse(p *models.TokenPair) tokenResponse {
	return tokenResponse{AccessToken: p.AccessToken, RefreshToken: p.RefreshToken}
}

type departmentRequest struct {
	Name string `json:"name"`
}

type departmentResponse struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func newDepartmentResponse(d *models.Department) departmentResponse {
	return departmentResponse{ID: d.ID, Name: d.Name, CreatedAt: d.CreatedAt, UpdatedAt: d.UpdatedAt}
}

type positionRequest struct {
	Name       string  `json:"name"`
	BaseSalary float64 `json:"base_salary"`
}

type positionResponse struct {
	ID         int64     `json:"id"`
	Name       string    `json:"name"`
	BaseSalary float64   `json:"base_salary"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func newPositionResponse(p *models.Position) positionResponse {
	return positionResponse{ID: p.ID, Name: p.Name, BaseSalary: p.BaseSalary, CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt}
}

type createEmployeeRequest struct {
	Email        string `json:"email"`
	Password     string `json:"password"`
	FirstName    string `json:"first_name"`
	LastName     string `json:"last_name"`
	Phone        string `json:"phone"`
	DepartmentID int64  `json:"department_id"`
	PositionID   int64  `json:"position_id"`
	HireDate     string `json:"hire_date"`
}

type updateEmployeeRequest struct {
	FirstName    string `json:"first_name"`
	LastName     string `json:"last_name"`
	Phone        string `json:"phone"`
	DepartmentID int64  `json:"department_id"`
	PositionID   int64  `json:"position_id"`
	Status       string `json:"status"`
}

type employeeResponse struct {
	ID              int64   `json:"id"`
	UserID          int64   `json:"user_id"`
	Email           string  `json:"email"`
	FirstName       string  `json:"first_name"`
	LastName        string  `json:"last_name"`
	Phone           string  `json:"phone"`
	DepartmentID    int64   `json:"department_id"`
	DepartmentName  string  `json:"department_name"`
	PositionID      int64   `json:"position_id"`
	PositionName    string  `json:"position_name"`
	HireDate        string  `json:"hire_date"`
	TerminationDate *string `json:"termination_date"`
	Status          string  `json:"status"`
}

func newEmployeeResponse(e *models.Employee) employeeResponse {
	r := employeeResponse{
		ID:             e.ID,
		UserID:         e.UserID,
		Email:          e.Email,
		FirstName:      e.FirstName,
		LastName:       e.LastName,
		Phone:          e.Phone,
		DepartmentID:   e.DepartmentID,
		DepartmentName: e.DepartmentName,
		PositionID:     e.PositionID,
		PositionName:   e.PositionName,
		HireDate:       formatDate(e.HireDate),
		Status:         e.Status,
	}
	if e.TerminationDate != nil {
		s := formatDate(*e.TerminationDate)
		r.TerminationDate = &s
	}
	return r
}

type employeePageResponse struct {
	Items []employeeResponse `json:"items"`
	Total int64              `json:"total"`
	Page  int                `json:"page"`
	Limit int                `json:"limit"`
}

func newEmployeePageResponse(p *models.EmployeePage) employeePageResponse {
	items := make([]employeeResponse, 0, len(p.Items))
	for i := range p.Items {
		items = append(items, newEmployeeResponse(&p.Items[i]))
	}
	return employeePageResponse{Items: items, Total: p.Total, Page: p.Page, Limit: p.Limit}
}

type meResponse struct {
	ID       int64             `json:"id"`
	Email    string            `json:"email"`
	Role     string            `json:"role"`
	Employee *employeeResponse `json:"employee"`
}

func newMeResponse(p *models.Profile) meResponse {
	r := meResponse{ID: p.UserID, Email: p.Email, Role: p.Role}
	if p.Employee != nil {
		e := newEmployeeResponse(p.Employee)
		r.Employee = &e
	}
	return r
}

type salaryRequest struct {
	Amount        float64 `json:"amount"`
	EffectiveDate string  `json:"effective_date"`
}

type salaryResponse struct {
	ID            int64     `json:"id"`
	EmployeeID    int64     `json:"employee_id"`
	Amount        float64   `json:"amount"`
	EffectiveDate string    `json:"effective_date"`
	CreatedAt     time.Time `json:"created_at"`
}

func newSalaryResponse(s *models.Salary) salaryResponse {
	return salaryResponse{ID: s.ID, EmployeeID: s.EmployeeID, Amount: s.Amount, EffectiveDate: formatDate(s.EffectiveDate), CreatedAt: s.CreatedAt}
}

type departmentReportResponse struct {
	DepartmentID   int64   `json:"department_id"`
	Department     string  `json:"department"`
	EmployeesCount int64   `json:"employees_count"`
	TotalSalary    float64 `json:"total_salary"`
	AvgSalary      float64 `json:"avg_salary"`
}

type timeOffRequest struct {
	LeaveType string `json:"leave_type"`
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
}

type timeOffDecisionRequest struct {
	Status string `json:"status"`
}

type timeOffResponse struct {
	ID         int64     `json:"id"`
	EmployeeID int64     `json:"employee_id"`
	ApprovedBy *int64    `json:"approved_by"`
	LeaveType  string    `json:"leave_type"`
	StartDate  string    `json:"start_date"`
	EndDate    string    `json:"end_date"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func newTimeOffResponse(t *models.TimeOff) timeOffResponse {
	return timeOffResponse{
		ID:         t.ID,
		EmployeeID: t.EmployeeID,
		ApprovedBy: t.ApprovedBy,
		LeaveType:  t.LeaveType,
		StartDate:  formatDate(t.StartDate),
		EndDate:    formatDate(t.EndDate),
		Status:     t.Status,
		CreatedAt:  t.CreatedAt,
		UpdatedAt:  t.UpdatedAt,
	}
}

type historyResponse struct {
	ID             int64   `json:"id"`
	EmployeeID     int64   `json:"employee_id"`
	DepartmentID   int64   `json:"department_id"`
	DepartmentName string  `json:"department_name"`
	PositionID     int64   `json:"position_id"`
	PositionName   string  `json:"position_name"`
	StartDate      string  `json:"start_date"`
	EndDate        *string `json:"end_date"`
}

func newHistoryResponse(h *models.PositionHistory) historyResponse {
	r := historyResponse{
		ID:             h.ID,
		EmployeeID:     h.EmployeeID,
		DepartmentID:   h.DepartmentID,
		DepartmentName: h.DepartmentName,
		PositionID:     h.PositionID,
		PositionName:   h.PositionName,
		StartDate:      formatDate(h.StartDate),
	}
	if h.EndDate != nil {
		s := formatDate(*h.EndDate)
		r.EndDate = &s
	}
	return r
}

type importRowErrorResponse struct {
	Row   int    `json:"row"`
	Error string `json:"error"`
}

type importResponse struct {
	Created int                      `json:"created"`
	Errors  []importRowErrorResponse `json:"errors"`
}

func newImportResponse(r *models.ImportResult) importResponse {
	errs := make([]importRowErrorResponse, 0, len(r.Errors))
	for _, e := range r.Errors {
		errs = append(errs, importRowErrorResponse{Row: e.Row, Error: e.Error})
	}
	return importResponse{Created: r.Created, Errors: errs}
}
