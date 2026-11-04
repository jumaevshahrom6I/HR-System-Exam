package models

const (
	RoleHR       = "hr"
	RoleEmployee = "employee"
)

type Role struct {
	ID   int64
	Code string
	Name string
}
