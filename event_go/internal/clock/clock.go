package clock

import "time"

// Clock 隔离业务代码对系统时间的依赖。
type Clock interface {
	Now() time.Time
}

type System struct{}

func (System) Now() time.Time {
	return time.Now()
}
