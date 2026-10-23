// Package cache berisi implementasi service.Cache: Redis untuk dijalankan,
// Noop untuk saat Redis tidak dikonfigurasi.
package cache

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"

	"tokoherbal/product-service/internal/breaker"
)

const (
	// Redis lokal menjawab dalam hitungan milidetik. 300 ms sudah sangat longgar;
	// lebih lama dari itu berarti Redis bermasalah dan lebih baik langsung ke database.
	opTimeout = 300 * time.Millisecond
	// DeletePrefix memakai beberapa putaran SCAN, jadi diberi waktu lebih panjang.
	scanTimeout = time.Second

	// Setelah breakerThreshold kegagalan berturut-turut, Redis dianggap mati dan
	// semua panggilan ditolak instan selama breakerCooldown, lalu dicoba lagi satu per satu.
	breakerThreshold = 3
	breakerCooldown  = 10 * time.Second
)

type Redis struct {
	rdb *redis.Client
	br  *breaker.Breaker
}

// NewRedis membuat klien dengan timeout pendek. Tanpa timeout, Redis yang mati
// atau lambat akan menahan setiap request, padahal cache hanyalah optimasi.
//
// Timeout bawaan klien saja TIDAK cukup: saat container Redis berhenti, nama
// host "redis" gagal di-resolve dan klien mengulang percobaan sambung belasan
// detik per perintah. Satu request memanggil Redis 2-3 kali, jadi total bisa
// melewati WriteTimeout server. Karena itu semua operasi dibungkus breaker.Do:
// batas waktu keras per operasi, plus penolakan instan saat Redis dianggap mati.
func NewRedis(addr string) *Redis {
	return &Redis{
		rdb: redis.NewClient(&redis.Options{
			Addr:         addr,
			DialTimeout:  time.Second,
			ReadTimeout:  500 * time.Millisecond,
			WriteTimeout: 500 * time.Millisecond,
			MaxRetries:   1,
		}),
		br: breaker.New(breakerThreshold, breakerCooldown),
	}
}

func (c *Redis) Close() error { return c.rdb.Close() }

func (c *Redis) Ping(ctx context.Context) error {
	return c.br.Do(ctx, opTimeout, func(ctx context.Context) error {
		return c.rdb.Ping(ctx).Err()
	})
}

func (c *Redis) Get(ctx context.Context, key string) ([]byte, bool, error) {
	var (
		val   []byte
		found bool
	)
	err := c.br.Do(ctx, opTimeout, func(ctx context.Context) error {
		v, err := c.rdb.Get(ctx, key).Bytes()
		if errors.Is(err, redis.Nil) {
			return nil // key tidak ada = cache miss, bukan kegagalan Redis
		}
		if err != nil {
			return err
		}
		val, found = v, true
		return nil
	})
	if err != nil {
		return nil, false, err
	}
	return val, found, nil
}

func (c *Redis) Set(ctx context.Context, key string, val []byte, ttl time.Duration) error {
	return c.br.Do(ctx, opTimeout, func(ctx context.Context) error {
		return c.rdb.Set(ctx, key, val, ttl).Err()
	})
}

// DeletePrefix memakai SCAN (bertahap, tidak memblokir Redis) dan bukan KEYS
// (memblokir seluruh server saat datanya banyak).
func (c *Redis) DeletePrefix(ctx context.Context, prefix string) error {
	return c.br.Do(ctx, scanTimeout, func(ctx context.Context) error {
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
	})
}

// Incr menambah penghitung untuk rate limit dan mengembalikan nilai barunya.
//
// SETNX (dengan TTL) dijalankan SEBELUM INCR dalam satu pipeline, sehingga key
// selalu punya masa kedaluwarsa. Versi "INCR lalu EXPIRE bila hasilnya 1" bisa
// meninggalkan key tanpa TTL jika proses mati di antara dua perintah itu, dan
// pengguna terblokir selamanya.
func (c *Redis) Incr(ctx context.Context, key string, window time.Duration) (int64, error) {
	var n int64
	err := c.br.Do(ctx, opTimeout, func(ctx context.Context) error {
		pipe := c.rdb.Pipeline()
		pipe.SetNX(ctx, key, 0, window)
		incr := pipe.Incr(ctx, key)
		if _, err := pipe.Exec(ctx); err != nil {
			return err
		}
		n = incr.Val()
		return nil
	})
	if err != nil {
		return 0, err
	}
	return n, nil
}
