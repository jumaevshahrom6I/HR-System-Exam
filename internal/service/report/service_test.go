package report_test

import (
	"context"
	"errors"
	"testing"

	"hr-system/internal/models"
	"hr-system/internal/repository/mocks"
	"hr-system/internal/service/report"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestDepartmentReport(t *testing.T) {
	ctx := context.Background()
	boom := errors.New("boom")

	t.Run("success", func(t *testing.T) {
		repo := mocks.NewMockSalaryRepository(t)
		repo.EXPECT().DepartmentReport(mock.Anything).Return([]models.DepartmentReportRow{{DepartmentID: 1, Department: "IT", EmployeesCount: 3, TotalSalary: 15000, AvgSalary: 5000}}, nil).Once()

		rows, err := report.NewService(repo).DepartmentReport(ctx)

		require.NoError(t, err)
		require.Equal(t, 5000.0, rows[0].AvgSalary)
	})

	t.Run("repository failure", func(t *testing.T) {
		repo := mocks.NewMockSalaryRepository(t)
		repo.EXPECT().DepartmentReport(mock.Anything).Return(nil, boom).Once()

		_, err := report.NewService(repo).DepartmentReport(ctx)

		require.ErrorIs(t, err, boom)
	})
}
