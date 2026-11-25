package redis

import (
	"context"
	"e-commerce/config"

	"log"

	"time"

	goredis "github.com/redis/go-redis/v9"
)

var Client *goredis.Client

func NewRedisConnection() {

	opts, err := goredis.ParseURL(config.RedisURL)
	if err != nil {
		log.Printf("[ERROR] ao analisar REDIS_URL: %v", err)

	}

	opts.DialTimeout = 5 * time.Second
	opts.ReadTimeout = 3 * time.Second
	opts.WriteTimeout = 3 * time.Second

	client := goredis.NewClient(opts)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		log.Fatal("[ERROR] falha ao se conectar com o Redis: ", err)
	}

	Client = client
}
