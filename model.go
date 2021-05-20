package redislock

import (
	"time"

	"github.com/go-redsync/redsync/v4"
)

// Locker 接口，分布式锁管理器，用于生成分布式锁
type Locker interface {
	GetMutex(key string, expire time.Duration, options ...redsync.Option) (Mutex, error)
}

// Mutex 分布式锁
type Mutex interface {
	Lock() error
	UnLock() error
	Valid() (bool, error)
}
