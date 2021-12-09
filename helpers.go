package redislock

import (
	"time"

	"github.com/pkg/errors"

	"github.com/go-redsync/redsync/v4"
)

func GetAndLock(locker Locker, key string, expire time.Duration, options ...redsync.Option) (mutex Mutex, err error) {
	if mutex, err = locker.GetMutex(key, expire, options...); err != nil {
		return nil, errors.Wrap(err, `生成锁失败`)
	}

	return mutex, mutex.Lock()
}
