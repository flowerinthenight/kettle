package kettle

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/gomodule/redigo/redis"
)

func NewRedisPool() (*redis.Pool, error) {
	addr := os.Getenv("REDIS_HOST")
	if addr == "" {
		return nil, fmt.Errorf("REDIS_HOST not set (host:port)")
	}

	var dialOpts []redis.DialOption
	password := os.Getenv("REDIS_PASSWORD")
	if password != "" {
		dialOpts = append(dialOpts, redis.DialPassword(password))
	}

	tm := os.Getenv("REDIS_TIMEOUT_SECONDS")
	if tm != "" {
		tmsec, err := strconv.Atoi(tm)
		if err != nil {
			return nil, fmt.Errorf("REDIS_TIMEOUT_SECONDS convert failed: %w", err)
		} else {
			dialOpts = append(dialOpts, redis.DialConnectTimeout(time.Duration(tmsec)*time.Second))
		}
	}

	rp := &redis.Pool{
		MaxIdle:     3,
		MaxActive:   4,
		Wait:        true,
		IdleTimeout: 240 * time.Second,
		Dial:        func() (redis.Conn, error) { return redis.Dial("tcp", addr, dialOpts...) },
	}

	return rp, nil
}
