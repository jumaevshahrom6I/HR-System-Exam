package department_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"hr-system/internal/models"
	"hr-system/internal/repository/mocks"
	"hr-system/internal/service/department"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var errBoom = errors.New("boom")

func newSvc(t *testing.T) (*department.Service, *mocks.MockDepartmentRepository) {
	t.Helper()
	repo := mocks.NewMockDepartmentRepository(t)
	return department.NewService(repo), repo
}

func TestCreate(t *testing.T) {
	ctx := context.Background()

	t.Run("success: name is trimmed", func(t *testing.T) {
		svc, repo := newSvc(t)
		repo.EXPECT().Create(mock.Anything, models.DepartmentInput{Name: "IT"}).Return(&models.Department{ID: 1, Name: "IT"}, nil).Once()

		got, err := svc.Create(ctx, models.DepartmentInput{Name: "  IT  "})

		require.NoError(t, err)
		require.Equal(t, "IT", got.Name)
	})

	for name, in := range map[string]models.DepartmentInput{
		"empty name":     {Name: ""},
		"spaces only":    {Name: "   "},
		"101 characters": {Name: strings.Repeat("Я", 101)},
	} {
		t.Run(name, func(t *testing.T) {
			svc, _ := newSvc(t)

			_, err := svc.Create(ctx, in)

			require.ErrorIs(t, err, models.ErrInvalidInput)
		})
	}

	t.Run("name already taken", func(t *testing.T) {
		svc, repo := newSvc(t)
		repo.EXPECT().Create(mock.Anything, mock.Anything).Return(nil, models.ErrConflict).Once()

		_, err := svc.Create(ctx, models.DepartmentInput{Name: "IT"})

		require.ErrorIs(t, err, models.ErrConflict)
	})
}

func TestGetListUpdateDelete(t *testing.T) {
	ctx := context.Background()

	t.Run("Get: not found", func(t *testing.T) {
		svc, repo := newSvc(t)
		repo.EXPECT().GetByID(mock.Anything, int64(9)).Return(nil, models.ErrNotFound).Once()

		_, err := svc.Get(ctx, 9)

		require.ErrorIs(t, err, models.ErrNotFound)
	})

	t.Run("List: success and failure", func(t *testing.T) {
		svc, repo := newSvc(t)
		repo.EXPECT().List(mock.Anything).Return([]models.Department{{ID: 1}}, nil).Once()
		repo.EXPECT().List(mock.Anything).Return(nil, errBoom).Once()

		items, err := svc.List(ctx)
		require.NoError(t, err)
		require.Len(t, items, 1)

		_, err = svc.List(ctx)
		require.ErrorIs(t, err, errBoom)
	})

	t.Run("Update: success, invalid name, not found", func(t *testing.T) {
		svc, repo := newSvc(t)
		repo.EXPECT().Update(mock.Anything, int64(1), models.DepartmentInput{Name: "HR"}).Return(&models.Department{ID: 1, Name: "HR"}, nil).Once()
		repo.EXPECT().Update(mock.Anything, int64(9), mock.Anything).Return(nil, models.ErrNotFound).Once()

		got, err := svc.Update(ctx, 1, models.DepartmentInput{Name: " HR "})
		require.NoError(t, err)
		require.Equal(t, "HR", got.Name)

		_, err = svc.Update(ctx, 1, models.DepartmentInput{Name: ""})
		require.ErrorIs(t, err, models.ErrInvalidInput)

		_, err = svc.Update(ctx, 9, models.DepartmentInput{Name: "HR"})
		require.ErrorIs(t, err, models.ErrNotFound)
	})

	t.Run("Delete: success, has employees, not found", func(t *testing.T) {
		svc, repo := newSvc(t)
		repo.EXPECT().Delete(mock.Anything, int64(1)).Return(nil).Once()
		repo.EXPECT().Delete(mock.Anything, int64(2)).Return(models.ErrConflict).Once()
		repo.EXPECT().Delete(mock.Anything, int64(3)).Return(models.ErrNotFound).Once()

		require.NoError(t, svc.Delete(ctx, 1))
		require.ErrorIs(t, svc.Delete(ctx, 2), models.ErrConflict)
		require.ErrorIs(t, svc.Delete(ctx, 3), models.ErrNotFound)
	})
}
