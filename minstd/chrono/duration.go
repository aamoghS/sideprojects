package chrono

import (
	"context"
	"time"
)

type Duration int64

const (
	Nanosecond  Duration = 1
	Microsecond          = 1000 * Nanosecond
	Millisecond          = 1000 * Microsecond
	Second               = 1000 * Millisecond
)

func FromStd(d time.Duration) Duration {
	return Duration(d)
}

func (d Duration) Std() time.Duration {
	return time.Duration(d)
}

func (d Duration) String() string {
	return d.Std().String()
}

func (d Duration) Nanoseconds() int64 {
	return int64(d)
}

func (d Duration) Milliseconds() int64 {
	return int64(d) / int64(Millisecond)
}

func (d Duration) Seconds() float64 {
	return float64(d) / float64(Second)
}

func Sleep(d Duration) {
	time.Sleep(d.Std())
}

func WithTimeout(parent context.Context, d Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(parent, d.Std())
}

func NowUnixMilli() int64 {
	return time.Now().UnixMilli()
}
