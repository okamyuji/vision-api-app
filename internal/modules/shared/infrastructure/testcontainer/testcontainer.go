package testcontainer

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/mysql"
	rediscontainer "github.com/testcontainers/testcontainers-go/modules/redis"
	"github.com/testcontainers/testcontainers-go/wait"
)

// RedisContainer Redisコンテナのラッパー
type RedisContainer struct {
	Container *rediscontainer.RedisContainer
	Host      string
	Port      string
}

// MySQLContainer MySQLコンテナのラッパー
type MySQLContainer struct {
	Container *mysql.MySQLContainer
	Host      string
	Port      string
	Database  string
	User      string
	Password  string
}

// StartRedis Redisコンテナを起動
func StartRedis(ctx context.Context, t *testing.T) (*RedisContainer, error) {
	t.Helper()

	container, err := rediscontainer.Run(ctx,
		"redis:7-alpine",
		testcontainers.WithWaitStrategy(
			wait.ForLog("Ready to accept connections").WithStartupTimeout(30*time.Second),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to start redis container: %w", err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get redis host: %w", err)
	}

	port, err := container.MappedPort(ctx, "6379")
	if err != nil {
		return nil, fmt.Errorf("failed to get redis port: %w", err)
	}

	return &RedisContainer{
		Container: container,
		Host:      host,
		Port:      port.Port(),
	}, nil
}

// StartMySQL MySQLコンテナを起動
func StartMySQL(ctx context.Context, t *testing.T) (*MySQLContainer, error) {
	t.Helper()

	const (
		database = "testdb"
		user     = "testuser"
		password = "testpass"
	)

	container, err := mysql.Run(ctx,
		"mysql:8.0",
		mysql.WithDatabase(database),
		mysql.WithUsername(user),
		mysql.WithPassword(password),
		testcontainers.WithWaitStrategy(
			wait.ForLog("port: 3306  MySQL Community Server").WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to start mysql container: %w", err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get mysql host: %w", err)
	}

	port, err := container.MappedPort(ctx, "3306")
	if err != nil {
		return nil, fmt.Errorf("failed to get mysql port: %w", err)
	}

	return &MySQLContainer{
		Container: container,
		Host:      host,
		Port:      port.Port(),
		Database:  database,
		User:      user,
		Password:  password,
	}, nil
}

// Close Redisコンテナを停止
func (r *RedisContainer) Close(ctx context.Context) error {
	if r.Container != nil {
		return r.Container.Terminate(ctx)
	}
	return nil
}

// Close MySQLコンテナを停止
func (m *MySQLContainer) Close(ctx context.Context) error {
	if m.Container != nil {
		return m.Container.Terminate(ctx)
	}
	return nil
}

// ConnectionString Redis接続文字列を取得
func (r *RedisContainer) ConnectionString() string {
	return fmt.Sprintf("%s:%s", r.Host, r.Port)
}

// ConnectionString MySQL接続文字列を取得
func (m *MySQLContainer) ConnectionString() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true",
		m.User, m.Password, m.Host, m.Port, m.Database)
}

// NewRedisClient Redisクライアントを作成
func (r *RedisContainer) NewRedisClient() *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr: r.ConnectionString(),
	})
}
