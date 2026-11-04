package models

import "time"

type PositionHistory struct {
	ID             int64
	EmployeeID     int64
	DepartmentID   int64
	DepartmentName string
	PositionID     int64
	PositionName   string
	StartDate      time.Time
	EndDate        *time.Time
	CreatedAt      time.Time
}

type ImportRowError struct {
	Row   int
	Error string
}

type ImportResult struct {
	Created int
	Errors  []ImportRowError
}
