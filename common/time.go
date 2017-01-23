package common

import (
	"time"
)

const DEFAULT_TIME_FROMAT = "2006-01-02 15:04:05"

const YYYYMMDD_TIME_FORMAT = "20060102 15:04:05"
const YMDH_FROMAT = "2006010215"
const Y_M_D_FROMAT = "2006-01-02"

var LOCAL_LOCATION = time.Now().Local().Location()

// 解析本地时间
func ParseLocalTime(t string) (time.Time, error) {
	return time.ParseInLocation(DEFAULT_TIME_FROMAT, t, LOCAL_LOCATION)
}

// 解析本地时间
func ParseLocatTimeWithFormat(format, t string) (time.Time, error) {
	return time.ParseInLocation(format, t, LOCAL_LOCATION)
}

// 取得毫秒
func UnixMills(t time.Time) int64 {
	return t.UnixNano() / int64(time.Millisecond)
}

// 取得毫秒
func UnixMillsTime(tmillis int64) time.Time {
	return time.Unix(tmillis/1000, (tmillis%1000)*int64(time.Millisecond))
}
