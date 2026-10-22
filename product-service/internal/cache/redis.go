// Package cache berisi implementasi service.Cache: Redis untuk dijalankan,
// Noop untuk saat Redis tidak dikonfigurasi.
package cache

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

type Redis struct {
	rdb *redis.Client
}

// NewRedis membuat klien dengan timeout pendek. Tanpa timeout, Redis yang mati
// atau lambat akan menahan setiap request, padahal cache hanyalah optimasi.
func NewRedis(addr string) *Redis {
	return &Redis{rdb: redis.NewClient(&redis.Options{
		Addr:         addr,
		DialTimeout:  time.Second,
		ReadTimeout:  500 * time.Millisecond,
		WriteTimeout: 500 * time.Millisecond,
		MaxRetries:   1,
	})}
}

func (c *Redis) Close() error { return c.rdb.Close() }

func (c *Redis) Ping(ctx context.Context) error {
	return c.rdb.Ping(ctx).Err()
}

func (c *Redis) Get(ctx context.Context, key string) ([]byte, bool, error) {
	val, err := c.rdb.Get(ctx, key).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, false, nil // key tidak ada = cache miss, bukan error
	}
	if err != nil {
		return nil, false, err
	}
	return val, true, nil
}

func (c *Redis) Set(ctx context.Context, key string, val []byte, ttl time.Duration) error {
	return c.rdb.Set(ctx, key, val, ttl).Err()
}

// DeletePrefix memakai SCAN (bertahap, tidak memblokir Redis) dan bukan KEYS
// (memblokir seluruh server saat datanya banyak).
func (c *Redis) DeletePrefix(ctx context.Context, prefix string) error {
	iter := c.rdb.Scan(ctx, 0, prefix+"*", 100).Iterator()
	var keys []string
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}
	if err := iter.Err(); err != nil {
		return err
	}
	if len(keys) == 0 {
		return nil
	}
	return c.rdb.Del(ctx, keys...).Err()
}

// Incr menambah penghitung untuk rate limit dan mengembalikan nilai barunya.
//
// SETNX (dengan TTL) dijalankan SEBELUM INCR dalam satu pipeline, sehingga key
// selalu punya masa kedaluwarsa. Versi "INCR lalu EXPIRE bila hasilnya 1" bisa
// meninggalkan key tanpa TTL jika proses mati di antara dua perintah itu, dan
// pengguna terblokir selamanya.
func (c *Redis) Incr(ctx context.Context, key string, window time.Duration) (int64, error) {
	pipe := c.rdb.Pipeline()
	pipe.SetNX(ctx, key, 0, window)
	incr := pipe.Incr(ctx, key)
	if _, err := pipe.Exec(ctx); err != nil {
		return 0, err
	}
	return incr.Val(), nil
}
