package position_test

import (
	"context"
	"testing"

	"hr-system/internal/models"
	"hr-system/internal/repository/mocks"
	"hr-system/internal/service/position"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func newSvc(t *testing.T) (*position.Service, *mocks.MockPositionRepository) {
	t.Helper()
	repo := mocks.NewMockPositionRepository(t)
	return position.NewService(repo), repo
}

func TestCreate(t *testing.T) {
	ctx := context.Background()

	t.Run("success: base salary 0 is allowed", func(t *testing.T) {
		svc, repo := newSvc(t)
		repo.EXPECT().Create(mock.Anything, models.PositionInput{Name: "Dev", BaseSalary: 0}).Return(&models.Position{ID: 1, Name: "Dev"}, nil).Once()

		got, err := svc.Create(ctx, models.PositionInput{Name: " Dev ", BaseSalary: 0})

		require.NoError(t, err)
		require.Equal(t, "Dev", got.Name)
	})

	for name, in := range map[string]models.PositionInput{
		"empty name":           {Name: " ", BaseSalary: 100},
		"negative base salary": {Name: "Dev", BaseSalary: -1},
	} {
		t.Run(name, func(t *testing.T) {
			svc, _ := newSvc(t)

			_, err := svc.Create(ctx, in)

			require.ErrorIs(t, err, models.ErrInvalidInput)
		})
	}

	t.Run("title already taken", func(t *testing.T) {
		svc, repo := newSvc(t)
		repo.EXPECT().Create(mock.Anything, mock.Anything).Return(nil, models.ErrConflict).Once()

		_, err := svc.Create(ctx, models.PositionInput{Name: "Dev", BaseSalary: 1})

		require.ErrorIs(t, err, models.ErrConflict)
	})
}

func TestGetListUpdateDelete(t *testing.T) {
	ctx := context.Background()
	svc, repo := newSvc(t)
	repo.EXPECT().GetByID(mock.Anything, int64(9)).Return(nil, models.ErrNotFound).Once()
	repo.EXPECT().List(mock.Anything).Return([]models.Position{{ID: 1}, {ID: 2}}, nil).Once()
	repo.EXPECT().Update(mock.Anything, int64(1), models.PositionInput{Name: "Lead", BaseSalary: 500}).Return(&models.Position{ID: 1, Name: "Lead"}, nil).Once()
	repo.EXPECT().Delete(mock.Anything, int64(1)).Return(models.ErrConflict).Once()

	_, err := svc.Get(ctx, 9)
	require.ErrorIs(t, err, models.ErrNotFound)

	items, err := svc.List(ctx)
	require.NoError(t, err)
	require.Len(t, items, 2)

	got, err := svc.Update(ctx, 1, models.PositionInput{Name: "Lead", BaseSalary: 500})
	require.NoError(t, err)
	require.Equal(t, "Lead", got.Name)

	_, err = svc.Update(ctx, 1, models.PositionInput{Name: "Lead", BaseSalary: -5})
	require.ErrorIs(t, err, models.ErrInvalidInput)

	require.ErrorIs(t, svc.Delete(ctx, 1), models.ErrConflict)
}
