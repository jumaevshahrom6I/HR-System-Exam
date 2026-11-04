package refresh

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"hr-system/internal/models"
	"hr-system/internal/repository"

	"github.com/redis/go-redis/v9"
)

const (
	tokenPrefix = "refresh:"
	userPrefix  = "refresh_user:"
)

type Repository struct {
	rdb *redis.Client
}

var _ repository.RefreshTokenRepository = (*Repository)(nil)

func NewRepository(rdb *redis.Client) *Repository {
	return &Repository{rdb: rdb}
}

func tokenKey(hash string) string { return tokenPrefix + hash }

func userKey(userID int64) string { return userPrefix + strconv.FormatInt(userID, 10) }

func (r *Repository) Create(ctx context.Context, tokenHash string, rt models.RefreshToken) error {
	ttl := time.Until(rt.ExpiresAt)
	if ttl <= 0 {
		return fmt.Errorf("%w: refresh token already expired", models.ErrInvalidInput)
	}

	pipe := r.rdb.TxPipeline()
	pipe.Set(ctx, tokenKey(tokenHash), strconv.FormatInt(rt.UserID, 10), ttl)
	pipe.SAdd(ctx, userKey(rt.UserID), tokenHash)
	pipe.Expire(ctx, userKey(rt.UserID), ttl)
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("save refresh token: %w", err)
	}
	return nil
}

func (r *Repository) Consume(ctx context.Context, tokenHash string) (int64, error) {
	val, err := r.rdb.GetDel(ctx, tokenKey(tokenHash)).Result()
	if errors.Is(err, redis.Nil) {
		return 0, models.ErrNotFound
	}
	if err != nil {
		return 0, fmt.Errorf("consume refresh token: %w", err)
	}

	userID, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse refresh token owner: %w", err)
	}

	_ = r.rdb.SRem(ctx, userKey(userID), tokenHash).Err()
	return userID, nil
}

func (r *Repository) Delete(ctx context.Context, tokenHash string) error {
	val, err := r.rdb.GetDel(ctx, tokenKey(tokenHash)).Result()
	if errors.Is(err, redis.Nil) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("delete refresh token: %w", err)
	}
	if userID, perr := strconv.ParseInt(val, 10, 64); perr == nil {
		_ = r.rdb.SRem(ctx, userKey(userID), tokenHash).Err()
	}
	return nil
}

func (r *Repository) RevokeAllForUser(ctx context.Context, userID int64) error {
	hashes, err := r.rdb.SMembers(ctx, userKey(userID)).Result()
	if err != nil {
		return fmt.Errorf("list user refresh tokens: %w", err)
	}

	keys := make([]string, 0, len(hashes)+1)
	for _, h := range hashes {
		keys = append(keys, tokenKey(h))
	}
	keys = append(keys, userKey(userID))
	if err := r.rdb.Del(ctx, keys...).Err(); err != nil {
		return fmt.Errorf("revoke refresh tokens: %w", err)
	}
	return nil
}
