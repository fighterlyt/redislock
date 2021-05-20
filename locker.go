package redislock

import (
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/go-redsync/redsync/v4"
	"github.com/go-redsync/redsync/v4/redis/goredis/v8"
)

var (
	// ErrEmptyLockKey  锁key为空
	ErrEmptyLockKey = errors.New("key不能为空")
)

// locker 基于redsync的redis分布式锁
type locker struct {
	redSync *redsync.Redsync
}

/*NewLocker 构建一个分布式锁管理器
参数:
*	redisClient	*redis.Client   redis客户端
返回值:
*	Locker	Locker
*/
func NewLocker(redisClient *redis.Client) Locker {
	return &locker{
		redSync: redsync.New(goredis.NewPool(redisClient)),
	}
}

/*GetMutex 获取分布式锁，key不能为空
参数:
*	key    	string                  分布式锁key
*	expire 	time.Duration           锁的有效期
*	options	...redsync.Option       锁其他参数
返回值:
*	Mutex	Mutex                   分布式锁
*	error	error                   错误
*/
func (l locker) GetMutex(key string, expire time.Duration, options ...redsync.Option) (Mutex, error) {
	if len(strings.TrimSpace(key)) == 0 {
		return nil, ErrEmptyLockKey
	}

	options = append(options, redsync.WithExpiry(expire))

	mutex := l.redSync.NewMutex(key, options...)
	return newMutex(mutex, expire), nil
}

// mutex 真实的mutex,实现了Mutex接口
type mutex struct {
	mutex  *redsync.Mutex // redsync Mutex
	expire time.Duration  // 超时时间,利用这个时间延期
	exit   chan struct{}  // 退出
	lock   *sync.Mutex    // 并发锁,拥有保护exit
}

/*newMutex 构建一个mutex
参数:
*	innerMutex	*redsync.Mutex
*	expire    	time.Duration
返回值:
*	*mutex	*mutex
*/
func newMutex(innerMutex *redsync.Mutex, expire time.Duration) *mutex {
	return &mutex{
		mutex:  innerMutex,
		expire: expire,
		exit:   make(chan struct{}),
		lock:   &sync.Mutex{},
	}
}

/*Lock 加锁
参数:
返回值:
*	error	error
说明:
	并发的go routine 启动之后，只会从某一个case 中return
	*   <-ticker.C 此时，会对锁进行延期，如果延期失败，会清理掉ticker,同时清理m.exit
	*   <-m.exit 此时，会清理ticker ,必然由m.UnLock 触发
		*   m.UnLock 会关闭m.exit
*/
func (m *mutex) Lock() error {
	// 锁
	if err := m.mutex.Lock(); err != nil {
		return err
	}
	// 自动续期
	go func() {
		// 选择一个略小于超时间隔的时间
		ticker := time.NewTicker(time.Duration(int64(m.expire) * 9 / 10))

		for {
			select {
			case <-m.exit: // 收到退出信号，关闭定时器
				ticker.Stop()
				return
			case <-ticker.C: // 到期,自动续期
				expired, err := m.mutex.Extend()
				if err != nil || expired { // 如果加锁失败，或者已经逾期，清理内部数据
					ticker.Stop()

					m.lock.Lock()

					if m.exit != nil {
						close(m.exit)
						m.exit = nil
					}

					m.lock.Unlock()

					return
				}
			}
		}
	}()

	return nil
}

/*UnLock 解锁
参数:
返回值:
*	error	error
*/
func (m *mutex) UnLock() error {
	m.lock.Lock()
	defer m.lock.Unlock()

	if m.exit != nil {
		close(m.exit)
		m.exit = nil
	}
	_, err := m.mutex.Unlock()

	return err
}

/*Valid 判断锁是否有效
参数:
返回值:
*	bool 	bool
*	error	error
*/
func (m *mutex) Valid() (bool, error) {
	return m.mutex.Valid()
}
