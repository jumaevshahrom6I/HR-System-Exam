package models

import (
	"fmt"
	"net/mail"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	StatusActive     = "active"
	StatusOnLeave    = "on_leave"
	StatusTerminated = "terminated"
)

type Employee struct {
	ID              int64
	UserID          int64
	Email           string
	FirstName       string
	LastName        string
	Phone           string
	DepartmentID    int64
	DepartmentName  string
	PositionID      int64
	PositionName    string
	HireDate        time.Time
	TerminationDate *time.Time
	Status          string
}

type EmployeeInput struct {
	Email        string
	Password     string
	FirstName    string
	LastName     string
	Phone        string
	DepartmentID int64
	PositionID   int64
	HireDate     time.Time
}

type EmployeeUpdate struct {
	FirstName    string
	LastName     string
	Phone        string
	DepartmentID int64
	PositionID   int64
	Status       string
}

type EmployeeFilter struct {
	Q            string
	DepartmentID *int64
	PositionID   *int64
	Status       string
	Page         int
	Limit        int
}

type EmployeePage struct {
	Items []Employee
	Total int64
	Page  int
	Limit int
}

type Profile struct {
	UserID   int64
	Email    string
	Role     string
	Employee *Employee
}

func validStatus(s string) bool {
	return s == StatusActive || s == StatusOnLeave || s == StatusTerminated
}

func checkNames(first, last, phone *string) error {
	*first = strings.TrimSpace(*first)
	*last = strings.TrimSpace(*last)
	*phone = strings.TrimSpace(*phone)
	if n := utf8.RuneCountInString(*first); n < 1 || n > 50 {
		return fmt.Errorf("%w: first_name must be 1-50 characters", ErrInvalidInput)
	}
	if n := utf8.RuneCountInString(*last); n < 1 || n > 50 {
		return fmt.Errorf("%w: last_name must be 1-50 characters", ErrInvalidInput)
	}
	if n := utf8.RuneCountInString(*phone); n < 1 || n > 20 {
		return fmt.Errorf("%w: phone must be 1-20 characters", ErrInvalidInput)
	}
	return nil
}

func (in *EmployeeInput) Validate() error {
	in.Email = strings.ToLower(strings.TrimSpace(in.Email))
	addr, err := mail.ParseAddress(in.Email)
	if err != nil || addr.Address != in.Email || len(in.Email) > 100 || !strings.Contains(in.Email[strings.LastIndex(in.Email, "@"):], ".") {
		return fmt.Errorf("%w: email is not valid", ErrInvalidInput)
	}
	if l := len(in.Password); l < 8 || l > 72 {
		return fmt.Errorf("%w: password must be 8-72 bytes", ErrInvalidInput)
	}
	if err := checkNames(&in.FirstName, &in.LastName, &in.Phone); err != nil {
		return err
	}
	if in.DepartmentID <= 0 || in.PositionID <= 0 {
		return fmt.Errorf("%w: department_id and position_id are required", ErrInvalidInput)
	}
	if in.HireDate.IsZero() {
		return fmt.Errorf("%w: hire_date is required", ErrInvalidInput)
	}
	return nil
}

func (in *EmployeeUpdate) Validate() error {
	if err := checkNames(&in.FirstName, &in.LastName, &in.Phone); err != nil {
		return err
	}
	if in.DepartmentID <= 0 || in.PositionID <= 0 {
		return fmt.Errorf("%w: department_id and position_id are required", ErrInvalidInput)
	}
	if !validStatus(in.Status) {
		return fmt.Errorf("%w: status must be active, on_leave or terminated", ErrInvalidInput)
	}
	return nil
}
