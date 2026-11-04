package models

import (
	"fmt"
	"time"
)

const (
	LeaveVacation = "vacation"
	LeaveSick     = "sick"

	TimeOffPending  = "pending"
	TimeOffApproved = "approved"
	TimeOffRejected = "rejected"
)

type TimeOff struct {
	ID         int64
	EmployeeID int64
	ApprovedBy *int64
	LeaveType  string
	StartDate  time.Time
	EndDate    time.Time
	Status     string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type TimeOffInput struct {
	LeaveType string
	StartDate time.Time
	EndDate   time.Time
}

func (in *TimeOffInput) Validate() error {
	if in.LeaveType != LeaveVacation && in.LeaveType != LeaveSick {
		return fmt.Errorf("%w: leave_type must be vacation or sick", ErrInvalidInput)
	}
	if in.StartDate.IsZero() || in.EndDate.IsZero() {
		return fmt.Errorf("%w: start_date and end_date are required", ErrInvalidInput)
	}
	if in.EndDate.Before(in.StartDate) {
		return fmt.Errorf("%w: end_date must not be before start_date", ErrInvalidInput)
	}
	return nil
}

type TimeOffFilter struct {
	Status     string
	EmployeeID *int64
}

func (f *TimeOffFilter) Validate() error {
	switch f.Status {
	case "", TimeOffPending, TimeOffApproved, TimeOffRejected:
		return nil
	}
	return fmt.Errorf("%w: status must be pending, approved or rejected", ErrInvalidInput)
}

func ValidateTimeOffDecision(status string) error {
	if status != TimeOffApproved && status != TimeOffRejected {
		return fmt.Errorf("%w: status must be approved or rejected", ErrInvalidInput)
	}
	return nil
}
