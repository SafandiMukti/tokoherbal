package cache

import (
	"context"
	"time"
)

// Noop dipakai bila REDIS_ADDR kosong: tidak menyimpan apa pun, semua Get = miss,
// dan tidak ada pembatasan rate. Aplikasi tetap berjalan penuh tanpa Redis.
type Noop struct{}

func (Noop) Get(context.Context, string) ([]byte, bool, error)          { return nil, false, nil }
func (Noop) Set(context.Context, string, []byte, time.Duration) error   { return nil }
func (Noop) DeletePrefix(context.Context, string) error                 { return nil }
func (Noop) Incr(context.Context, string, time.Duration) (int64, error) { return 0, nil }
