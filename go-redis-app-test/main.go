package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/kelseyhightower/envconfig"
)

// Config contains the configuration for the app.
type Config struct {
	Port          int    `envconfig:"PORT"`
	RedisHost     string `envconfig:"REDIS_HOST"`
	RedisPort     int    `envconfig:"REDIS_PORT"`
	RedisPassword string `envconfig:"REDIS_PASSWORD"`
	RedisDb       int    `envconfig:"REDIS_DB"`
}

// Count is the response object.
type Count struct {
	Value int `json:"count"`
}

func main() {
	var c Config
	err := envconfig.Process("app", &c)
	if err != nil {
		log.Fatal(err.Error())
	}

	r := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", c.RedisHost, c.RedisPort),
		Password: c.RedisPassword,
		DB:       c.RedisDb,
	})

	router := gin.Default()

	router.GET("/:id", func(c *gin.Context) {
		ctx := context.Background()
		id := c.Param("id")
		val, err := r.Get(ctx, id).Result()
		if err != nil {
			log.Printf("redis get error %v ", err)
			if err == redis.Nil {
				c.JSON(http.StatusOK, Count{Value: 1})
				if err := r.Set(ctx, id, 1, 0).Err(); err != nil {
					log.Printf("failed to update cache %v", err)
				}
				return
			}
			c.String(http.StatusInternalServerError, "failed to get count")
		}
		ct, err := strconv.Atoi(val)
		if err != nil {
			log.Printf("failed to parse value %v", err)
		}

		c.JSON(http.StatusOK, Count{Value: ct + 1})
		if err := r.Set(ctx, id, ct+1, 0).Err(); err != nil {
			log.Printf("failed to update cache  %v", err)
		}
		return
	})

	router.GET("/ping", func(c *gin.Context) {
		if _, err := r.Ping(context.Background()).Result(); err != nil {
			log.Printf("ping error %v", err)
			c.Status(http.StatusInternalServerError)
		} else {
			c.Status(http.StatusOK)
		}
	})

	_ = router.Run(fmt.Sprintf(":%d", c.Port))
}
