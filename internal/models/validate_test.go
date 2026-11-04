package models

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func validEmployeeInput() EmployeeInput {
	return EmployeeInput{
		Email: "ivan@company.local", Password: "Pass12345",
		FirstName: "Ivan", LastName: "Petrov", Phone: "+992900000000",
		DepartmentID: 1, PositionID: 1, HireDate: time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
	}
}

func TestLoginInputValidate(t *testing.T) {
	in := LoginInput{Email: "  HR@Company.Local ", Password: " keep spaces "}
	if err := in.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if in.Email != "hr@company.local" || in.Password != " keep spaces " {
		t.Fatalf("got %+v: email must be normalized, password must stay as is", in)
	}
	for name, bad := range map[string]LoginInput{
		"empty email":    {Email: "  ", Password: "x"},
		"empty password": {Email: "a@b.c", Password: ""},
	} {
		if err := bad.Validate(); !errors.Is(err, ErrInvalidInput) {
			t.Fatalf("%s: error = %v, want ErrInvalidInput", name, err)
		}
	}
}

func TestEmployeeInputValidate(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*EmployeeInput)
		ok     bool
	}{
		{"valid", func(*EmployeeInput) {}, true},
		{"email normalized", func(in *EmployeeInput) { in.Email = "  Ivan@Company.Local " }, true},
		{"name in cyrillic 50 chars", func(in *EmployeeInput) { in.FirstName = strings.Repeat("Я", 50) }, true},
		{"password 8 bytes", func(in *EmployeeInput) { in.Password = "12345678" }, true},
		{"password 72 bytes", func(in *EmployeeInput) { in.Password = strings.Repeat("a", 72) }, true},
		{"phone 20 chars", func(in *EmployeeInput) { in.Phone = strings.Repeat("1", 20) }, true},
		{"not an email", func(in *EmployeeInput) { in.Email = "not-an-email" }, false},
		{"email without dot in domain", func(in *EmployeeInput) { in.Email = "a@localhost" }, false},
		{"empty email", func(in *EmployeeInput) { in.Email = "" }, false},
		{"password 7 bytes", func(in *EmployeeInput) { in.Password = "1234567" }, false},
		{"password 73 bytes", func(in *EmployeeInput) { in.Password = strings.Repeat("a", 73) }, false},
		{"empty first name", func(in *EmployeeInput) { in.FirstName = "" }, false},
		{"first name of spaces", func(in *EmployeeInput) { in.FirstName = "   " }, false},
		{"first name 51 chars", func(in *EmployeeInput) { in.FirstName = strings.Repeat("a", 51) }, false},
		{"empty last name", func(in *EmployeeInput) { in.LastName = "" }, false},
		{"empty phone", func(in *EmployeeInput) { in.Phone = "" }, false},
		{"phone 21 chars", func(in *EmployeeInput) { in.Phone = strings.Repeat("1", 21) }, false},
		{"no department", func(in *EmployeeInput) { in.DepartmentID = 0 }, false},
		{"no position", func(in *EmployeeInput) { in.PositionID = 0 }, false},
		{"no hire date", func(in *EmployeeInput) { in.HireDate = time.Time{} }, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := validEmployeeInput()
			tt.mutate(&in)
			err := in.Validate()
			if tt.ok && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !tt.ok && !errors.Is(err, ErrInvalidInput) {
				t.Fatalf("error = %v, want ErrInvalidInput", err)
			}
		})
	}
	in := validEmployeeInput()
	in.Email = "  Ivan@Company.Local "
	_ = in.Validate()
	if in.Email != "ivan@company.local" {
		t.Fatalf("email = %q, want normalized", in.Email)
	}
}

func TestEmployeeUpdateValidate(t *testing.T) {
	valid := EmployeeUpdate{FirstName: "Ivan", LastName: "Petrov", Phone: "1", DepartmentID: 1, PositionID: 1, Status: "active"}
	for _, status := range []string{"active", "on_leave", "terminated"} {
		in := valid
		in.Status = status
		if err := in.Validate(); err != nil {
			t.Fatalf("status %q: unexpected error: %v", status, err)
		}
	}
	bad := map[string]func(*EmployeeUpdate){
		"unknown status": func(in *EmployeeUpdate) { in.Status = "weird" },
		"empty status":   func(in *EmployeeUpdate) { in.Status = "" },
		"empty name":     func(in *EmployeeUpdate) { in.FirstName = " " },
		"empty phone":    func(in *EmployeeUpdate) { in.Phone = "" },
		"no department":  func(in *EmployeeUpdate) { in.DepartmentID = 0 },
	}
	for name, mutate := range bad {
		in := valid
		mutate(&in)
		if err := in.Validate(); !errors.Is(err, ErrInvalidInput) {
			t.Fatalf("%s: error = %v, want ErrInvalidInput", name, err)
		}
	}
}

func TestDepartmentAndPositionValidate(t *testing.T) {
	d := DepartmentInput{Name: "  IT  "}
	if err := d.Validate(); err != nil || d.Name != "IT" {
		t.Fatalf("got (%v, %q), want trimmed name and no error", err, d.Name)
	}
	long := DepartmentInput{Name: strings.Repeat("Я", 101)}
	if err := long.Validate(); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("101 characters: error = %v, want ErrInvalidInput", err)
	}
	if err := (&DepartmentInput{Name: "   "}).Validate(); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("empty name: error = %v, want ErrInvalidInput", err)
	}

	if err := (&PositionInput{Name: "Dev", BaseSalary: 0}).Validate(); err != nil {
		t.Fatalf("zero base salary must be allowed: %v", err)
	}
	if err := (&PositionInput{Name: "Dev", BaseSalary: -1}).Validate(); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("negative base salary: error = %v, want ErrInvalidInput", err)
	}
}

func TestSalaryInputValidate(t *testing.T) {
	date := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	if err := (&SalaryInput{Amount: 5000, EffectiveDate: date}).Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for name, in := range map[string]SalaryInput{
		"zero amount":     {Amount: 0, EffectiveDate: date},
		"negative amount": {Amount: -5, EffectiveDate: date},
		"no date":         {Amount: 5000},
	} {
		if err := in.Validate(); !errors.Is(err, ErrInvalidInput) {
			t.Fatalf("%s: error = %v, want ErrInvalidInput", name, err)
		}
	}
}

func TestCurrentUserIsHR(t *testing.T) {
	if !(CurrentUser{ID: 1, Role: "hr"}).IsHR() || (CurrentUser{ID: 2, Role: "employee"}).IsHR() {
		t.Fatalf("IsHR must be true only for role hr")
	}
}

func TestTimeOffInputValidate(t *testing.T) {
	start := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		name string
		in   TimeOffInput
		ok   bool
	}{
		{"vacation", TimeOffInput{LeaveType: "vacation", StartDate: start, EndDate: start.AddDate(0, 0, 13)}, true},
		{"sick leave for one day", TimeOffInput{LeaveType: "sick", StartDate: start, EndDate: start}, true},
		{"unknown type", TimeOffInput{LeaveType: "weekend", StartDate: start, EndDate: start}, false},
		{"empty type", TimeOffInput{StartDate: start, EndDate: start}, false},
		{"end before start", TimeOffInput{LeaveType: "vacation", StartDate: start, EndDate: start.AddDate(0, 0, -1)}, false},
		{"no start date", TimeOffInput{LeaveType: "vacation", EndDate: start}, false},
		{"no end date", TimeOffInput{LeaveType: "vacation", StartDate: start}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.in.Validate()
			if tt.ok && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !tt.ok && !errors.Is(err, ErrInvalidInput) {
				t.Fatalf("error = %v, want ErrInvalidInput", err)
			}
		})
	}
}

func TestTimeOffFilterAndDecision(t *testing.T) {
	for _, status := range []string{"", "pending", "approved", "rejected"} {
		f := TimeOffFilter{Status: status}
		if err := f.Validate(); err != nil {
			t.Fatalf("status %q: unexpected error: %v", status, err)
		}
	}
	f := TimeOffFilter{Status: "weird"}
	if err := f.Validate(); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("unknown status: error = %v", err)
	}

	for _, status := range []string{"approved", "rejected"} {
		if err := ValidateTimeOffDecision(status); err != nil {
			t.Fatalf("decision %q: %v", status, err)
		}
	}
	for _, status := range []string{"pending", "", "weird"} {
		if err := ValidateTimeOffDecision(status); !errors.Is(err, ErrInvalidInput) {
			t.Fatalf("decision %q must be rejected, error = %v", status, err)
		}
	}
}
