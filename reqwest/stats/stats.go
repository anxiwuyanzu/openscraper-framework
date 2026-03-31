package stats

import (
	"fmt"
	"github.com/anxiwuyanzu/openscraper-framework/spider-common-go/v4/dot"
	"sync"
	"sync/atomic"
	"time"
)

var (
	myStats *stats
	once    = sync.Once{}
)

func init() {
	once.Do(func() {
		myStats = newStats()
	})
}

type stats struct {
	acquireProxies        uint32
	dialProxies           uint32
	dialFailedProxies     uint32
	expiredProxies        uint32
	acquireProxiesTimeout uint32
	releaseProxies        uint32
	closedProxies         uint32

	request       uint32
	requestFailed uint32
}

func newStats() *stats {
	s := &stats{}
	go s.tick()
	return s
}

func (s *stats) tick() {
	if dot.Debug() {
		return
	}
	for range time.Tick(5 * time.Minute) {
		// 这样记录后, 在日志库可以方便将state 加起来, 然后做告警, 画图...
		dot.Logger().WithField("state", s.acquireProxies).Info("reqwest stats acquireProxies")
		dot.Logger().WithField("state", s.dialProxies).Info("reqwest stats dialProxies")
		dot.Logger().WithField("state", s.dialFailedProxies).Info("reqwest stats dialFailedProxies")
		dot.Logger().WithField("state", s.expiredProxies).Info("reqwest stats expiredProxies")
		dot.Logger().WithField("state", s.acquireProxiesTimeout).Info("reqwest stats acquireProxiesTimeout")
		dot.Logger().WithField("state", s.releaseProxies).Info("reqwest stats releaseProxies")
		dot.Logger().WithField("state", s.closedProxies).Info("reqwest stats closedProxies")

		dot.Logger().WithField("state", s.requestFailed).Info("reqwest stats requestFailed")
		dot.Logger().WithField("state", s.request).Info("reqwest stats request")

		atomic.StoreUint32(&myStats.acquireProxies, 0)
		atomic.StoreUint32(&myStats.dialProxies, 0)
		atomic.StoreUint32(&myStats.dialFailedProxies, 0)
		atomic.StoreUint32(&myStats.expiredProxies, 0)
		atomic.StoreUint32(&myStats.acquireProxiesTimeout, 0)
		atomic.StoreUint32(&myStats.releaseProxies, 0)
		atomic.StoreUint32(&myStats.closedProxies, 0)
		atomic.StoreUint32(&myStats.requestFailed, 0)
		atomic.StoreUint32(&myStats.request, 0)
	}
}

func (s *stats) String() string {
	return fmt.Sprintf("acquire: %d, acquire_timeout: %d, dial: %d, dial_failed; %d, expired: %d, release: %d",
		s.acquireProxies, s.acquireProxiesTimeout, s.dialProxies, s.dialFailedProxies, s.expiredProxies, s.releaseProxies)
}

func IncrAcquireProxies(n uint32) {
	atomic.AddUint32(&myStats.acquireProxies, n)
}

func IncrDialProxies() {
	atomic.AddUint32(&myStats.dialProxies, 1)
}

func IncrDialFailedProxies() {
	atomic.AddUint32(&myStats.dialFailedProxies, 1)
}

func IncrExpiredProxies() {
	atomic.AddUint32(&myStats.expiredProxies, 1)
}

func IncrAcquireProxiesTimeout() {
	atomic.AddUint32(&myStats.acquireProxiesTimeout, 1)
}

func IncrReleaseProxies() {
	atomic.AddUint32(&myStats.expiredProxies, 1)
}

func IncrClosedProxies() {
	atomic.AddUint32(&myStats.closedProxies, 1)
}

func IncrRequest() {
	atomic.AddUint32(&myStats.request, 1)
}

func IncrRequestFailed() {
	atomic.AddUint32(&myStats.requestFailed, 1)
}
