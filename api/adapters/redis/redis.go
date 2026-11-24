package redis

import (
	"context"
	"e-commerce/config"
	"log"
	"net"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

var Client *goredis.Client

func NewRedisConnection() {
	addr := net.JoinHostPort(config.RedisHost, config.RedisPort)

	client := goredis.NewClient(&goredis.Options{
		Addr:         addr,
		Username:     config.RedisUser,
		Password:     config.RedisPassword,
		DB:           0,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		log.Fatal("[ERROR] falha ao se conectar com o Redis: ", err)
	}

	Client = client
}
