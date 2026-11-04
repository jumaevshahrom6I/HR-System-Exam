package handler_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"hr-system/internal/handler"
	v1 "hr-system/internal/handler/v1"
	"hr-system/internal/models"
	"hr-system/internal/service"
	"hr-system/internal/service/mocks"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	m.Run()
}

const (
	hrToken       = "hr-token"
	employeeToken = "employee-token"
)

type env struct {
	engine     *gin.Engine
	auth       *mocks.MockAuthService
	department *mocks.MockDepartmentService
	position   *mocks.MockPositionService
	employee   *mocks.MockEmployeeService
	salary     *mocks.MockSalaryService
	report     *mocks.MockReportService
	timeoff    *mocks.MockTimeOffService
	pingErr    error
}

func newEnv(t *testing.T) *env {
	t.Helper()
	e := &env{
		auth:       mocks.NewMockAuthService(t),
		department: mocks.NewMockDepartmentService(t),
		position:   mocks.NewMockPositionService(t),
		employee:   mocks.NewMockEmployeeService(t),
		salary:     mocks.NewMockSalaryService(t),
		report:     mocks.NewMockReportService(t),
		timeoff:    mocks.NewMockTimeOffService(t),
	}
	e.auth.EXPECT().Authenticate(hrToken).Return(&models.AccessClaims{UserID: 1, Role: "hr"}, nil).Maybe()
	e.auth.EXPECT().Authenticate(employeeToken).Return(&models.AccessClaims{UserID: 50, Role: "employee"}, nil).Maybe()
	e.auth.EXPECT().Authenticate(mock.AnythingOfType("string")).Return(nil, models.ErrInvalidToken).Maybe()

	services := &service.Services{
		Auth: e.auth, Department: e.department, Position: e.position,
		Employee: e.employee, Salary: e.salary, Report: e.report, TimeOff: e.timeoff,
	}
	checks := []v1.Check{{Name: "postgres", Ping: func(context.Context) error { return e.pingErr }}}
	e.engine = handler.NewHandler(v1.NewHandler(services, checks), e.auth, "http://localhost:8081").InitRoutes()
	return e
}

func (e *env) do(method, path, token, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	e.engine.ServeHTTP(w, req)
	return w
}

func TestHealth(t *testing.T) {
	t.Run("200 when dependencies respond", func(t *testing.T) {
		e := newEnv(t)

		w := e.do("GET", "/health", "", "")

		require.Equal(t, http.StatusOK, w.Code)
		require.JSONEq(t, `{"status":"ok"}`, w.Body.String())
	})

	t.Run("503 when a dependency is down", func(t *testing.T) {
		e := newEnv(t)
		e.pingErr = errors.New("down")

		w := e.do("GET", "/health", "", "")

		require.Equal(t, http.StatusServiceUnavailable, w.Code)
		require.JSONEq(t, `{"error":"postgres is unavailable"}`, w.Body.String())
	})
}

func TestAuthorization(t *testing.T) {
	tests := []struct {
		name   string
		token  string
		header string
		want   int
		body   string
	}{
		{"no header", "", "", http.StatusUnauthorized, `{"error":"missing or invalid token"}`},
		{"scheme is not Bearer", "", "Basic abc", http.StatusUnauthorized, `{"error":"missing or invalid token"}`},
		{"empty token", "", "Bearer ", http.StatusUnauthorized, `{"error":"missing or invalid token"}`},
		{"bad token", "garbage", "", http.StatusUnauthorized, `{"error":"invalid token"}`},
		{"employee on an HR route", employeeToken, "", http.StatusForbidden, `{"error":"forbidden"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := newEnv(t)
			req := httptest.NewRequest("GET", "/api/v1/departments", nil)
			if tt.token != "" {
				req.Header.Set("Authorization", "Bearer "+tt.token)
			}
			if tt.header != "" {
				req.Header.Set("Authorization", tt.header)
			}
			w := httptest.NewRecorder()

			e.engine.ServeHTTP(w, req)

			require.Equal(t, tt.want, w.Code)
			require.JSONEq(t, tt.body, w.Body.String())
		})
	}
}

func TestLogin(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		e := newEnv(t)
		e.auth.EXPECT().Login(mock.Anything, models.LoginInput{Email: "hr@company.local", Password: "Pass12345"}).
			Return(&models.TokenPair{AccessToken: "a", RefreshToken: "r"}, nil).Once()

		w := e.do("POST", "/api/v1/auth/login", "", `{"email":"hr@company.local","password":"Pass12345"}`)

		require.Equal(t, http.StatusOK, w.Code)
		require.JSONEq(t, `{"access_token":"a","refresh_token":"r"}`, w.Body.String())
	})

	t.Run("invalid credentials: 401", func(t *testing.T) {
		e := newEnv(t)
		e.auth.EXPECT().Login(mock.Anything, mock.Anything).Return(nil, models.ErrInvalidCredentials).Once()

		w := e.do("POST", "/api/v1/auth/login", "", `{"email":"x@y.z","password":"bad"}`)

		require.Equal(t, http.StatusUnauthorized, w.Code)
		require.JSONEq(t, `{"error":"invalid email or password"}`, w.Body.String())
	})

	bad := map[string]string{
		"extra field":    `{"email":"a","password":"b","role":"hr"}`,
		"malformed JSON": `{"email":`,
		"empty body":     ``,
		"not an object":  `[1,2]`,
		"body too large": `{"email":"` + strings.Repeat("a", 2<<20) + `"}`,
	}
	for name, body := range bad {
		t.Run("400: "+name, func(t *testing.T) {
			e := newEnv(t)

			w := e.do("POST", "/api/v1/auth/login", "", body)

			require.Equal(t, http.StatusBadRequest, w.Code)
			require.JSONEq(t, `{"error":"invalid request body"}`, w.Body.String())
		})
	}

	t.Run("validation error reaches the client without service prefixes", func(t *testing.T) {
		e := newEnv(t)
		e.auth.EXPECT().Login(mock.Anything, mock.Anything).
			Return(nil, fmt.Errorf("%w: password is required", models.ErrInvalidInput)).Once()

		w := e.do("POST", "/api/v1/auth/login", "", `{"email":"a@b.c","password":""}`)

		require.Equal(t, http.StatusBadRequest, w.Code)
		require.JSONEq(t, `{"error":"password is required"}`, w.Body.String())
	})
}

func TestRefreshAndLogout(t *testing.T) {
	e := newEnv(t)
	e.auth.EXPECT().Refresh(mock.Anything, "old").Return(&models.TokenPair{AccessToken: "a2", RefreshToken: "r2"}, nil).Once()
	e.auth.EXPECT().Refresh(mock.Anything, "used").Return(nil, models.ErrInvalidToken).Once()
	e.auth.EXPECT().Logout(mock.Anything, "r2").Return(nil).Once()

	w := e.do("POST", "/api/v1/auth/refresh", "", `{"refresh_token":"old"}`)
	require.Equal(t, http.StatusOK, w.Code)
	require.JSONEq(t, `{"access_token":"a2","refresh_token":"r2"}`, w.Body.String())

	w = e.do("POST", "/api/v1/auth/refresh", "", `{"refresh_token":"used"}`)
	require.Equal(t, http.StatusUnauthorized, w.Code)

	w = e.do("POST", "/api/v1/auth/logout", "", `{"refresh_token":"r2"}`)
	require.Equal(t, http.StatusNoContent, w.Code)
	require.Empty(t, w.Body.String())
}

func TestDepartments(t *testing.T) {
	t.Run("create: 201", func(t *testing.T) {
		e := newEnv(t)
		e.department.EXPECT().Create(mock.Anything, models.DepartmentInput{Name: "IT"}).Return(&models.Department{ID: 1, Name: "IT"}, nil).Once()

		w := e.do("POST", "/api/v1/departments", hrToken, `{"name":"IT"}`)

		require.Equal(t, http.StatusCreated, w.Code)
		require.Contains(t, w.Body.String(), `"id":1`)
		require.Contains(t, w.Body.String(), `"name":"IT"`)
	})

	t.Run("empty list is [] not null", func(t *testing.T) {
		e := newEnv(t)
		e.department.EXPECT().List(mock.Anything).Return(nil, nil).Once()

		w := e.do("GET", "/api/v1/departments", hrToken, "")

		require.Equal(t, http.StatusOK, w.Code)
		require.JSONEq(t, `[]`, w.Body.String())
	})

	for _, id := range []string{"abc", "0", "-5", "1.5"} {
		t.Run("invalid id "+id+": 400", func(t *testing.T) {
			e := newEnv(t)

			w := e.do("GET", "/api/v1/departments/"+id, hrToken, "")

			require.Equal(t, http.StatusBadRequest, w.Code)
			require.JSONEq(t, `{"error":"invalid id"}`, w.Body.String())
		})
	}

	t.Run("not found: 404", func(t *testing.T) {
		e := newEnv(t)
		e.department.EXPECT().Get(mock.Anything, int64(9)).Return(nil, models.ErrNotFound).Once()

		w := e.do("GET", "/api/v1/departments/9", hrToken, "")

		require.Equal(t, http.StatusNotFound, w.Code)
		require.JSONEq(t, `{"error":"not found"}`, w.Body.String())
	})

	t.Run("delete with employees: 409", func(t *testing.T) {
		e := newEnv(t)
		e.department.EXPECT().Delete(mock.Anything, int64(1)).
			Return(fmt.Errorf("delete department: %w", fmt.Errorf("%w: related records conflict", models.ErrConflict))).Once()

		w := e.do("DELETE", "/api/v1/departments/1", hrToken, "")

		require.Equal(t, http.StatusConflict, w.Code)
		require.JSONEq(t, `{"error":"related records conflict"}`, w.Body.String())
	})

	t.Run("delete: 204 without a body", func(t *testing.T) {
		e := newEnv(t)
		e.department.EXPECT().Delete(mock.Anything, int64(1)).Return(nil).Once()

		w := e.do("DELETE", "/api/v1/departments/1", hrToken, "")

		require.Equal(t, http.StatusNoContent, w.Code)
		require.Empty(t, w.Body.String())
	})

	t.Run("internal error: 500 without details", func(t *testing.T) {
		e := newEnv(t)
		e.department.EXPECT().Get(mock.Anything, int64(1)).Return(nil, errors.New("connection refused to 10.0.0.5")).Once()

		w := e.do("GET", "/api/v1/departments/1", hrToken, "")

		require.Equal(t, http.StatusInternalServerError, w.Code)
		require.JSONEq(t, `{"error":"internal server error"}`, w.Body.String())
	})

	t.Run("panic in a handler: 500, server keeps working", func(t *testing.T) {
		e := newEnv(t)
		e.department.EXPECT().List(mock.Anything).Run(func(context.Context) { panic("boom") }).Return(nil, nil).Once()
		e.department.EXPECT().Get(mock.Anything, int64(1)).Return(&models.Department{ID: 1}, nil).Once()

		w := e.do("GET", "/api/v1/departments", hrToken, "")
		require.Equal(t, http.StatusInternalServerError, w.Code)
		require.JSONEq(t, `{"error":"internal server error"}`, w.Body.String())

		w = e.do("GET", "/api/v1/departments/1", hrToken, "")
		require.Equal(t, http.StatusOK, w.Code)
	})
}

func TestEmployees(t *testing.T) {
	t.Run("list: defaults page=1 limit=20", func(t *testing.T) {
		e := newEnv(t)
		e.employee.EXPECT().List(mock.Anything, mock.MatchedBy(func(f models.EmployeeFilter) bool {
			return f.Page == 1 && f.Limit == 20 && f.Q == "iv" && f.Status == "active" &&
				f.DepartmentID != nil && *f.DepartmentID == 3 && f.PositionID == nil
		})).Return(&models.EmployeePage{Items: nil, Total: 0, Page: 1, Limit: 20}, nil).Once()

		w := e.do("GET", "/api/v1/employees?q=iv&status=active&department_id=3", hrToken, "")

		require.Equal(t, http.StatusOK, w.Code)
		require.JSONEq(t, `{"items":[],"total":0,"page":1,"limit":20}`, w.Body.String())
	})

	for _, q := range []string{"page=abc", "limit=x", "department_id=-1", "position_id=abc"} {
		t.Run("400 on "+q, func(t *testing.T) {
			e := newEnv(t)

			w := e.do("GET", "/api/v1/employees?"+q, hrToken, "")

			require.Equal(t, http.StatusBadRequest, w.Code)
		})
	}

	t.Run("create: hire date is parsed", func(t *testing.T) {
		e := newEnv(t)
		e.employee.EXPECT().Create(mock.Anything, mock.MatchedBy(func(in models.EmployeeInput) bool {
			return in.Email == "ivan@company.local" && in.HireDate.Format("2006-01-02") == "2026-01-15" && in.DepartmentID == 1
		})).Return(&models.Employee{ID: 5, Status: "active", HireDate: mustDate("2026-01-15")}, nil).Once()

		w := e.do("POST", "/api/v1/employees", hrToken,
			`{"email":"ivan@company.local","password":"Pass12345","first_name":"Ivan","last_name":"Petrov","phone":"+1","department_id":1,"position_id":1,"hire_date":"2026-01-15"}`)

		require.Equal(t, http.StatusCreated, w.Code)
		require.Contains(t, w.Body.String(), `"hire_date":"2026-01-15"`)
		require.Contains(t, w.Body.String(), `"termination_date":null`)
	})

	t.Run("create: invalid date 400", func(t *testing.T) {
		e := newEnv(t)

		w := e.do("POST", "/api/v1/employees", hrToken, `{"email":"a@b.c","hire_date":"15.01.2026"}`)

		require.Equal(t, http.StatusBadRequest, w.Code)
		require.JSONEq(t, `{"error":"hire_date must be in YYYY-MM-DD format"}`, w.Body.String())
	})

	t.Run("email already taken: 409", func(t *testing.T) {
		e := newEnv(t)
		e.employee.EXPECT().Create(mock.Anything, mock.Anything).Return(nil, fmt.Errorf("create employee: %w", fmt.Errorf("%w: already exists", models.ErrConflict))).Once()

		w := e.do("POST", "/api/v1/employees", hrToken, `{"email":"a@b.c","hire_date":"2026-01-15"}`)

		require.Equal(t, http.StatusConflict, w.Code)
		require.JSONEq(t, `{"error":"already exists"}`, w.Body.String())
	})

	t.Run("termination: 204", func(t *testing.T) {
		e := newEnv(t)
		e.employee.EXPECT().Delete(mock.Anything, int64(5)).Return(nil).Once()

		require.Equal(t, http.StatusNoContent, e.do("DELETE", "/api/v1/employees/5", hrToken, "").Code)
	})

	t.Run("/me is available to an employee", func(t *testing.T) {
		e := newEnv(t)
		e.employee.EXPECT().Me(mock.Anything, models.CurrentUser{ID: 50, Role: "employee"}).
			Return(&models.Profile{UserID: 50, Email: "ivan@company.local", Role: "employee"}, nil).Once()

		w := e.do("GET", "/api/v1/me", employeeToken, "")

		require.Equal(t, http.StatusOK, w.Code)
		require.JSONEq(t, `{"id":50,"email":"ivan@company.local","role":"employee","employee":null}`, w.Body.String())
	})

	t.Run("history: 200", func(t *testing.T) {
		e := newEnv(t)
		e.employee.EXPECT().History(mock.Anything, int64(5)).Return([]models.PositionHistory{{ID: 1, EmployeeID: 5, StartDate: mustDate("2026-01-15")}}, nil).Once()

		w := e.do("GET", "/api/v1/employees/5/history", hrToken, "")

		require.Equal(t, http.StatusOK, w.Code)
		require.Contains(t, w.Body.String(), `"end_date":null`)
	})
}

func TestRoutesDoNotConflict(t *testing.T) {
	e := newEnv(t)
	e.employee.EXPECT().Get(mock.Anything, int64(5)).Return(&models.Employee{ID: 5, HireDate: mustDate("2026-01-15")}, nil).Once()
	e.employee.EXPECT().Export(mock.Anything, mock.Anything, mock.Anything).
		Run(func(_ context.Context, _ models.EmployeeFilter, w io.Writer) {
			_, _ = w.Write([]byte("\xEF\xBB\xBFid,email\n"))
		}).Return(nil).Once()

	w := e.do("GET", "/api/v1/employees/5", hrToken, "")
	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), `"id":5`)

	w = e.do("GET", "/api/v1/employees/export.csv?status=active", hrToken, "")
	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "text/csv; charset=utf-8", w.Header().Get("Content-Type"))
	require.Contains(t, w.Header().Get("Content-Disposition"), "employees.csv")
	require.True(t, strings.HasPrefix(w.Body.String(), "\xEF\xBB\xBF"))
}

func TestImport(t *testing.T) {
	upload := func(t *testing.T, e *env, token, content string) *httptest.ResponseRecorder {
		t.Helper()
		var body bytes.Buffer
		mw := multipart.NewWriter(&body)
		part, err := mw.CreateFormFile("file", "employees.csv")
		require.NoError(t, err)
		_, _ = part.Write([]byte(content))
		require.NoError(t, mw.Close())
		req := httptest.NewRequest("POST", "/api/v1/employees/import", &body)
		req.Header.Set("Content-Type", mw.FormDataContentType())
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		e.engine.ServeHTTP(w, req)
		return w
	}

	t.Run("success: created and errors", func(t *testing.T) {
		e := newEnv(t)
		e.employee.EXPECT().Import(mock.Anything, mock.Anything).
			Return(&models.ImportResult{Created: 2, Errors: []models.ImportRowError{{Row: 3, Error: "email already registered"}}}, nil).Once()

		w := upload(t, e, hrToken, "email,password\n")

		require.Equal(t, http.StatusOK, w.Code)
		require.JSONEq(t, `{"created":2,"errors":[{"row":3,"error":"email already registered"}]}`, w.Body.String())
	})

	t.Run("no errors: errors is []", func(t *testing.T) {
		e := newEnv(t)
		e.employee.EXPECT().Import(mock.Anything, mock.Anything).Return(&models.ImportResult{Created: 1}, nil).Once()

		w := upload(t, e, hrToken, "x")

		require.JSONEq(t, `{"created":1,"errors":[]}`, w.Body.String())
	})

	t.Run("file larger than 1 MB: 400", func(t *testing.T) {
		e := newEnv(t)

		w := upload(t, e, hrToken, strings.Repeat("a", 1<<20+10))

		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("no file: 400", func(t *testing.T) {
		e := newEnv(t)

		w := e.do("POST", "/api/v1/employees/import", hrToken, `{}`)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("employee cannot import: 403", func(t *testing.T) {
		e := newEnv(t)

		w := upload(t, e, employeeToken, "x")

		require.Equal(t, http.StatusForbidden, w.Code)
	})
}

func TestTimeOffAndSalaries(t *testing.T) {
	t.Run("employee creates a request: 201", func(t *testing.T) {
		e := newEnv(t)
		e.timeoff.EXPECT().Create(mock.Anything, models.CurrentUser{ID: 50, Role: "employee"}, mock.MatchedBy(func(in models.TimeOffInput) bool {
			return in.LeaveType == "vacation" && in.StartDate.Format("2006-01-02") == "2026-07-01"
		})).Return(&models.TimeOff{ID: 1, EmployeeID: 5, LeaveType: "vacation", Status: "pending", StartDate: mustDate("2026-07-01"), EndDate: mustDate("2026-07-14")}, nil).Once()

		w := e.do("POST", "/api/v1/time-off", employeeToken, `{"leave_type":"vacation","start_date":"2026-07-01","end_date":"2026-07-14"}`)

		require.Equal(t, http.StatusCreated, w.Code)
		require.Contains(t, w.Body.String(), `"status":"pending"`)
		require.Contains(t, w.Body.String(), `"approved_by":null`)
	})

	t.Run("overlapping dates: 409", func(t *testing.T) {
		e := newEnv(t)
		e.timeoff.EXPECT().Create(mock.Anything, mock.Anything, mock.Anything).
			Return(nil, fmt.Errorf("create time off request: %w", fmt.Errorf("%w: dates overlap with another request", models.ErrConflict))).Once()

		w := e.do("POST", "/api/v1/time-off", employeeToken, `{"leave_type":"vacation","start_date":"2026-07-01","end_date":"2026-07-14"}`)

		require.Equal(t, http.StatusConflict, w.Code)
		require.JSONEq(t, `{"error":"dates overlap with another request"}`, w.Body.String())
	})

	t.Run("request decision: PATCH is HR only", func(t *testing.T) {
		e := newEnv(t)
		e.timeoff.EXPECT().UpdateStatus(mock.Anything, models.CurrentUser{ID: 1, Role: "hr"}, int64(3), "approved").
			Return(&models.TimeOff{ID: 3, Status: "approved", StartDate: mustDate("2026-07-01"), EndDate: mustDate("2026-07-14")}, nil).Once()

		require.Equal(t, http.StatusForbidden, e.do("PATCH", "/api/v1/time-off/3/status", employeeToken, `{"status":"approved"}`).Code)
		w := e.do("PATCH", "/api/v1/time-off/3/status", hrToken, `{"status":"approved"}`)
		require.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("request list: filters from query", func(t *testing.T) {
		e := newEnv(t)
		e.timeoff.EXPECT().List(mock.Anything, models.CurrentUser{ID: 1, Role: "hr"}, mock.MatchedBy(func(f models.TimeOffFilter) bool {
			return f.Status == "pending" && f.EmployeeID != nil && *f.EmployeeID == 5
		})).Return(nil, nil).Once()

		w := e.do("GET", "/api/v1/time-off?status=pending&employee_id=5", hrToken, "")

		require.Equal(t, http.StatusOK, w.Code)
		require.JSONEq(t, `[]`, w.Body.String())
	})

	t.Run("salary: 201 and invalid date 400", func(t *testing.T) {
		e := newEnv(t)
		e.salary.EXPECT().Add(mock.Anything, int64(5), mock.Anything).Return(&models.Salary{ID: 1, EmployeeID: 5, Amount: 5000, EffectiveDate: mustDate("2026-02-01")}, nil).Once()

		w := e.do("POST", "/api/v1/employees/5/salaries", hrToken, `{"amount":5000,"effective_date":"2026-02-01"}`)
		require.Equal(t, http.StatusCreated, w.Code)

		w = e.do("POST", "/api/v1/employees/5/salaries", hrToken, `{"amount":5000,"effective_date":"yesterday"}`)
		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("my salaries and report", func(t *testing.T) {
		e := newEnv(t)
		e.salary.EXPECT().ListMine(mock.Anything, models.CurrentUser{ID: 50, Role: "employee"}).Return([]models.Salary{}, nil).Once()
		e.report.EXPECT().DepartmentReport(mock.Anything).Return([]models.DepartmentReportRow{{DepartmentID: 1, Department: "IT", EmployeesCount: 2, TotalSalary: 10000, AvgSalary: 5000}}, nil).Once()

		require.JSONEq(t, `[]`, e.do("GET", "/api/v1/me/salaries", employeeToken, "").Body.String())
		require.Equal(t, http.StatusForbidden, e.do("GET", "/api/v1/reports/departments", employeeToken, "").Code)
		w := e.do("GET", "/api/v1/reports/departments", hrToken, "")
		require.JSONEq(t, `[{"department_id":1,"department":"IT","employees_count":2,"total_salary":10000,"avg_salary":5000}]`, w.Body.String())
	})
}

func TestCORSAndRequestID(t *testing.T) {
	e := newEnv(t)

	req := httptest.NewRequest("OPTIONS", "/api/v1/departments", nil)
	req.Header.Set("Origin", "http://localhost:8081")
	w := httptest.NewRecorder()
	e.engine.ServeHTTP(w, req)
	require.Equal(t, http.StatusNoContent, w.Code)
	require.Equal(t, "http://localhost:8081", w.Header().Get("Access-Control-Allow-Origin"))
	require.Contains(t, w.Header().Get("Access-Control-Allow-Headers"), "Authorization")

	req = httptest.NewRequest("GET", "/health", nil)
	req.Header.Set("Origin", "http://evil.example")
	req.Header.Set("X-Request-ID", "client-supplied")
	w = httptest.NewRecorder()
	e.engine.ServeHTTP(w, req)
	require.Empty(t, w.Header().Get("Access-Control-Allow-Origin"))
	require.NotEmpty(t, w.Header().Get("X-Request-ID"))
	require.NotEqual(t, "client-supplied", w.Header().Get("X-Request-ID"))
}
