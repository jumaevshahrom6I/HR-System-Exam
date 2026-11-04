package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"hr-system/internal/config"
	"hr-system/internal/handler"
	"hr-system/internal/handler/v1"
	"hr-system/internal/repository"
	repodepartment "hr-system/internal/repository/department"
	repoemployee "hr-system/internal/repository/employee"
	repohistory "hr-system/internal/repository/history"
	repoposition "hr-system/internal/repository/position"
	reporefresh "hr-system/internal/repository/refresh"
	reporole "hr-system/internal/repository/role"
	reposalary "hr-system/internal/repository/salary"
	repotimeoff "hr-system/internal/repository/timeoff"
	repouser "hr-system/internal/repository/user"
	"hr-system/internal/service"
	svcauth "hr-system/internal/service/auth"
	svcdepartment "hr-system/internal/service/department"
	svcemployee "hr-system/internal/service/employee"
	svcposition "hr-system/internal/service/position"
	svcreport "hr-system/internal/service/report"
	svcsalary "hr-system/internal/service/salary"
	svctimeoff "hr-system/internal/service/timeoff"
	"hr-system/pkg/logger"
	"hr-system/pkg/postgres"
	"hr-system/pkg/redis"

	"github.com/gin-gonic/gin"
)

const (
	swaggerOrigin = "http://localhost:8081"
	minJWTSecret  = 32
)

func Run(cfg *config.Config) error {
	if len(cfg.JWT.Secret) < minJWTSecret {
		return fmt.Errorf("JWT_SECRET must be at least %d bytes long", minJWTSecret)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := postgres.NewPool(ctx, postgres.Config{
		Host:     cfg.Postgres.Host,
		Port:     cfg.Postgres.Port,
		User:     cfg.Postgres.User,
		Password: cfg.Postgres.Password,
		Database: cfg.Postgres.Database,
		SSLMode:  cfg.Postgres.SSLMode,
	})
	if err != nil {
		return fmt.Errorf("connect to postgres: %w", err)
	}
	defer pool.Close()

	rdb, err := redis.NewClient(ctx, redis.Config{
		Host:     cfg.Redis.Host,
		Port:     cfg.Redis.Port,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	if err != nil {
		return fmt.Errorf("connect to redis: %w", err)
	}
	defer rdb.Close()

	repos := repository.Repositories{
		User:       repouser.NewRepository(pool),
		Role:       reporole.NewRepository(pool),
		Refresh:    reporefresh.NewRepository(rdb),
		Department: repodepartment.NewRepository(pool),
		Position:   repoposition.NewRepository(pool),
		Employee:   repoemployee.NewRepository(pool),
		Salary:     reposalary.NewRepository(pool),
		TimeOff:    repotimeoff.NewRepository(pool),
		History:    repohistory.NewRepository(pool),
	}

	services := &service.Services{
		Auth: svcauth.NewService(repos.User, repos.Refresh, svcauth.Config{
			Secret:     cfg.JWT.Secret,
			AccessTTL:  cfg.JWT.AccessTTL,
			RefreshTTL: cfg.JWT.RefreshTTL,
		}),
		Department: svcdepartment.NewService(repos.Department),
		Position:   svcposition.NewService(repos.Position),
		Employee:   svcemployee.NewService(repos.Employee, repos.User, repos.Role, repos.Department, repos.Position, repos.Refresh, repos.History),
		Salary:     svcsalary.NewService(repos.Salary, repos.Employee),
		Report:     svcreport.NewService(repos.Salary),
		TimeOff:    svctimeoff.NewService(repos.TimeOff, repos.Employee),
	}

	checks := []v1.Check{
		{Name: "postgres", Ping: pool.Ping},
		{Name: "redis", Ping: func(ctx context.Context) error { return rdb.Ping(ctx).Err() }},
	}
	if cfg.App.Env == logger.Prod {
		gin.SetMode(gin.ReleaseMode)
	}
	engine := handler.NewHandler(v1.NewHandler(services, checks), services.Auth, swaggerOrigin).InitRoutes()

	srv := &http.Server{
		Handler:      engine,
		ReadTimeout:  cfg.HTTP.ReadTimeout,
		WriteTimeout: cfg.HTTP.WriteTimeout,
	}

	ln, err := net.Listen("tcp", ":"+cfg.HTTP.Port)
	if err != nil {
		return fmt.Errorf("listen on port %s: %w", cfg.HTTP.Port, err)
	}

	serveErr := make(chan error, 1)
	go func() {
		slog.Info("server started", "addr", ln.Addr().String(), "env", cfg.App.Env)
		if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serveErr <- err
		}
	}()

	select {
	case err := <-serveErr:
		return fmt.Errorf("http server: %w", err)
	case <-ctx.Done():
	}

	slog.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.HTTP.ShutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown server: %w", err)
	}
	return nil
}
