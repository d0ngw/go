package common

import (
	"time"
)

const (
	// FormatDefault 默认的日期时间格式
	FormatDefault = "2006-01-02 15:04:05"
	// FormatYYYYMMDD 日期时间格式
	FormatYYYYMMDD = "20060102 15:04:05"
	// FormatYMDH 日期格式
	FormatYMDH = "2006010215"
	// FormatYMD 日期格式
	FormatYMD = "2006-01-02"
)

// LocalLocation 本地时区
var LocalLocation = time.Now().Local().Location()

// ParseLocalTime  解析本地时间
func ParseLocalTime(t string) (time.Time, error) {
	return time.ParseInLocation(FormatDefault, t, LocalLocation)
}

// ParseLocatTimeWithFormat 解析本地时间
func ParseLocatTimeWithFormat(format, t string) (time.Time, error) {
	return time.ParseInLocation(format, t, LocalLocation)
}

// UnixMills 取得毫秒
func UnixMills(t time.Time) int64 {
	return t.UnixNano() / int64(time.Millisecond)
}

// UnixMillsTime 根据毫秒取得时间
func UnixMillsTime(tmillis int64) time.Time {
	return time.Unix(tmillis/1000, (tmillis%1000)*int64(time.Millisecond))
}
