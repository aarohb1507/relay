package redis

import (
	"context"
	"log"

	goredis "github.com/redis/go-redis/v9"
)

var (
	Client *goredis.Client
	Ctx = context.Background()
)


func Connect() {

	Client = goredis.NewClient(&goredis.Options{
		Addr: "localhost:6379",
	})

	err := Client.Ping(Ctx).Err()
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Connected to Redis")
}

