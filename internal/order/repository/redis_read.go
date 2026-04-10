package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/parthasarathi/go-grpc-http/internal/order/model"
)

type redisReadRepo struct {
	client *redis.Client
}

func NewRedisReadRepository(client *redis.Client) ReadRepository {
	return &redisReadRepo{client: client}
}

func (r *redisReadRepo) GetByID(ctx context.Context, orderID string) (*model.OrderView, error) {
	data, err := r.client.Get(ctx, orderKey(orderID)).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("redis get: %w", err)
	}

	var view model.OrderView
	if err := json.Unmarshal(data, &view); err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}
	return &view, nil
}

func (r *redisReadRepo) GetByUserID(ctx context.Context, userID string) ([]*model.OrderView, error) {
	// Get order IDs from the user's sorted set
	orderIDs, err := r.client.ZRevRange(ctx, userOrdersKey(userID), 0, -1).Result()
	if err != nil {
		return nil, fmt.Errorf("redis zrevrange: %w", err)
	}

	var views []*model.OrderView
	for _, id := range orderIDs {
		view, err := r.GetByID(ctx, id)
		if err != nil {
			return nil, err
		}
		if view != nil {
			views = append(views, view)
		}
	}
	return views, nil
}

func (r *redisReadRepo) Save(ctx context.Context, view *model.OrderView) error {
	data, err := json.Marshal(view)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	pipe := r.client.Pipeline()

	// Store the order view
	pipe.Set(ctx, orderKey(view.OrderID), data, 24*time.Hour)

	// Maintain a sorted set of order IDs per user (score = unix timestamp)
	pipe.ZAdd(ctx, userOrdersKey(view.UserID), redis.Z{
		Score:  float64(view.CreatedAt.Unix()),
		Member: view.OrderID,
	})

	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("redis pipeline: %w", err)
	}
	return nil
}

func orderKey(orderID string) string {
	return "order:" + orderID
}

func userOrdersKey(userID string) string {
	return "user_orders:" + userID
}
