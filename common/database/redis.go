package database

import (
	"common/config"
	"common/logs"
	"context"
	"github.com/redis/go-redis/v9"
	"time"
)

type RedisManager struct {
	Client        *redis.Client        // 单机
	ClusterClient *redis.ClusterClient // 集群
}

func NewRedis() *RedisManager {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var (
		clusterClient *redis.ClusterClient
		client        *redis.Client
	)

	addrs := config.Conf.Database.RedisConf.ClusterAddrs
	if len(addrs) == 0 {
		client = redis.NewClient(&redis.Options{
			Addr:         config.Conf.Database.RedisConf.Addr,
			PoolSize:     config.Conf.Database.RedisConf.PoolSize,
			MinIdleConns: config.Conf.Database.RedisConf.MinIdleConns,
			Password:     config.Conf.Database.RedisConf.Password,
		})
	} else {
		clusterClient = redis.NewClusterClient(&redis.ClusterOptions{
			Addrs:        addrs,
			PoolSize:     config.Conf.Database.RedisConf.PoolSize,
			MinIdleConns: config.Conf.Database.RedisConf.MinIdleConns,
			Password:     config.Conf.Database.RedisConf.Password,
		})
	}

	if clusterClient != nil {
		if err := clusterClient.Ping(ctx).Err(); err != nil {
			logs.Fatal("redis cluster connect err: %v", err)
			return nil
		}
	}

	if client != nil {
		if err := client.Ping(ctx).Err(); err != nil {
			logs.Fatal("redis client connect err: %v", err)
			return nil
		}
	}

	return &RedisManager{
		Client:        client,
		ClusterClient: clusterClient,
	}
}

func (r *RedisManager) Close() {
	if r.ClusterClient != nil {
		if err := r.ClusterClient.Close(); err != nil {
			logs.Error("redis cluster client close err: %v", err)
		}
	}

	if r.Client != nil {
		if err := r.Client.Close(); err != nil {
			logs.Error("redis client close err: %v", err)
		}
	}
}

func (r *RedisManager) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	if r.ClusterClient != nil {
		return r.ClusterClient.Set(ctx, key, value, ttl).Err()
	}

	if r.Client != nil {
		return r.Client.Set(ctx, key, value, ttl).Err()
	}

	return nil
}
