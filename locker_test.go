package redislock

import (
	"sync"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/go-redsync/redsync/v4"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
)

func Test_mutex_Lock(t *testing.T) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     "localhost:9736",
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	locker := NewLocker(rdb)

	duration := time.Second

	mutex, err := locker.GetMutex(`test`, duration)
	require.NoError(t, err)

	require.NoError(t, mutex.Lock(), `加锁`)

	wg := &sync.WaitGroup{}

	wg.Add(1)

	go func() {
		var (
			newMutex Mutex
		)

		newMutex, err = locker.GetMutex(`test`, duration, redsync.WithRetryDelay(time.Millisecond*10))

		for {
			if err = newMutex.Lock(); err == nil {
				t.Log(`获取到锁`, time.Now().Format(`2006-01-02 15:04:05.000`))
				wg.Done()

				return
			}

			if !errors.Is(err, redsync.ErrFailed) {
				require.Error(t, err)
				return
			}

			t.Log(`未获取到锁`, time.Now().Format(`2006-01-02 15:04:05.000`))

			time.Sleep(duration / 2)
		}
	}()

	time.Sleep(duration * 3)
	t.Log(`释放`, time.Now().Format(`2006-01-02 15:04:05.000`))

	require.NoError(t, mutex.UnLock())
	wg.Wait()
}
