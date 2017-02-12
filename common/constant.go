package common

// Status 定义状态
type Status int8

const (
	// DISABLE 禁用
	DISABLE Status = 0
	// ENABLE 有效
	ENABLE Status = 1
)

// IsValid 判断状态是否是有效的
func (p Status) IsValid() bool {
	return p == DISABLE || p == ENABLE
}
