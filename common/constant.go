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

// Value 状态的值
func (p Status) Value() int8 {
	return int8(p)
}

// 定义环境变量
const (
	EnvWorkfDir = "work_dir"
)

// 定义环境的常量
const (
	EnvDev        = "dev"
	EnvTest       = "test"
	EnvProduction = "prod"
)
