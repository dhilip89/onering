package onering

import (
	"sync/atomic"
)

type MPSC struct {
	multi
}

func (r *MPSC) init(size uint32) {
	r.multi.init(size)
	r.rc = 1
}

func (r *MPSC) Get(i interface{}) bool {
	var (
		rp        = r.rc
		data, seq = r.frame(rp)
	)
	for ; rp > atomic.LoadInt64(seq); r.wait() {
		if rp > r.rp {
			atomic.StoreInt64(&r.rp, rp)
		} else if r.Done() {
			return false
		}
	}
	inject(i, *data)
	*seq = -rp
	r.rc = rp + 1
	if r.rc-r.rp > r.maxbatch {
		atomic.StoreInt64(&r.rp, r.rc)
	}
	return true
}

func (r *MPSC) Consume(i interface{}) {
	var (
		fn       = extractfn(i)
		maxbatch = int(r.maxbatch)
		it       iter
	)
	for keep := true; keep; {
		var rp, wp = r.rc, atomic.LoadInt64(&r.wp)
		for ; rp >= wp; r.wait() {
			if rp > r.rp {
				atomic.StoreInt64(&r.rp, r.rc)
			} else if r.Done() {
				return
			}
			wp = atomic.LoadInt64(&r.wp)
		}

		for i := 0; rp < wp && keep; it.inc() {
			var data, seq = r.frame(rp)
			if i++; atomic.LoadInt64(seq) <= 0 || i&maxbatch == 0 {
				r.rc = rp
				atomic.StoreInt64(&r.rp, rp)
				for atomic.LoadInt64(seq) <= 0 {
					r.wait()
				}
			}
			fn(&it, *data)
			*seq = -rp
			keep = !it.stop
			rp++
		}
		r.rc = rp
		atomic.StoreInt64(&r.rp, r.rc)
	}
}

func (r *MPSC) Put(i interface{}) {
	var wp = r.next(&r.wp)
	for diff := wp - r.mask; diff >= atomic.LoadInt64(&r.rp); {
		r.wait()
	}
	var pos = wp & r.mask
	r.data[pos] = extractptr(i)
	atomic.StoreInt64(&r.seq[pos], wp)
}
