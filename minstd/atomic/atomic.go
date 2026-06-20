package atomic

import stdatomic "sync/atomic"

type Int64 struct {
	v int64
}

func (i *Int64) Load() int64 {
	return stdatomic.LoadInt64(&i.v)
}

func (i *Int64) Store(v int64) {
	stdatomic.StoreInt64(&i.v, v)
}

func AddInt32(addr *int32, delta int32) int32 {
	return stdatomic.AddInt32(addr, delta)
}

func AddUint64(addr *uint64, delta uint64) uint64 {
	return stdatomic.AddUint64(addr, delta)
}

func LoadInt32(addr *int32) int32 {
	return stdatomic.LoadInt32(addr)
}

func LoadUint64(addr *uint64) uint64 {
	return stdatomic.LoadUint64(addr)
}

func CompareAndSwapUint32(addr *uint32, old, new uint32) bool {
	return stdatomic.CompareAndSwapUint32(addr, old, new)
}
