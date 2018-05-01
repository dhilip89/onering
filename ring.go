package onering

import (
	"math/bits"
	"runtime"
	"sync/atomic"
)

const MaxBatch = 255
const spin = 512 - 1 // not used at the moment

type ring struct {
	_    [8]int64
	wp   int64
	_    [7]int64
	rp   int64
	_    [7]int64
	data []int64
	mask int64
	done int32
}

func (r *ring) Init(size uint32) {
	r.data = make([]int64, 1<<uint32(32-bits.LeadingZeros32(size-1)))
	r.mask = int64(len(r.data) - 1)
}

func (r *ring) Close() {
	atomic.StoreInt32(&r.done, 1)
}

func (r *ring) Done() bool {
	return atomic.LoadInt32(&r.done) > 0 && atomic.LoadInt64(&r.wp) <= atomic.LoadInt64(&r.rp)
}

func (r *ring) wait() {
	runtime.Gosched()
}

type multi struct {
	ring
	size int64
	seq  []int64
	_    [50]byte
}

func (c *multi) Init(size uint32) {
	c.ring.Init(size)
	c.size = int64(len(c.data))
	c.seq = make([]int64, len(c.data))
	c.wp = 1 // just to avoid 0-awkwardness with seq
	c.rp = 1
}

func (c *multi) next(p *int64) int64 {
	return atomic.AddInt64(p, 1) - 1
}

func (c *multi) contents(p int64) (data, seq *int64) {
	var pos = c.mask & p
	return &c.data[pos], &c.seq[pos]
}

// empty sync.Locker for conditionals
type NoLock struct{}

func (NoLock) Lock()   {}
func (NoLock) Unlock() {}
