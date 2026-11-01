package common

import (
	"time"
)

const (
	// FormatDefault is the default date and time layout.
	FormatDefault = "2006-01-02 15:04:05"
	// FormatYYYYMMDD is a compact date layout.
	FormatYYYYMMDD = "20060102 15:04:05"
	// FormatYMDH is a date and hour layout.
	FormatYMDH = "2006010215"
	// FormatYMD is a date layout.
	FormatYMD = "2006-01-02"
)

// LocalLocation is the local time zone.
var LocalLocation = time.Now().Local().Location()

// ParseLocalTime parses a timestamp in the local time zone.
func ParseLocalTime(t string) (time.Time, error) {
	return time.ParseInLocation(FormatDefault, t, LocalLocation)
}

// ParseLocatTimeWithFormat parses a timestamp with the supplied layout in the local time zone.
func ParseLocatTimeWithFormat(format, t string) (time.Time, error) {
	return time.ParseInLocation(format, t, LocalLocation)
}

// UnixMills returns the Unix timestamp in milliseconds.
func UnixMills(t time.Time) int64 {
	return t.UnixNano() / int64(time.Millisecond)
}

// UnixMillsTime converts a Unix timestamp in milliseconds to a time.
func UnixMillsTime(tmillis int64) time.Time {
	return time.Unix(tmillis/1000, (tmillis%1000)*int64(time.Millisecond))
}
