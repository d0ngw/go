package common

// Status represents an enabled or disabled state.
type Status int8

const (
	// DISABLE marks a disabled state.
	DISABLE Status = 0
	// ENABLE marks an enabled state.
	ENABLE Status = 1
)

// IsValid reports whether the status is valid.
func (p Status) IsValid() bool {
	return p == DISABLE || p == ENABLE
}

// Value returns the status value.
func (p Status) Value() int8 {
	return int8(p)
}

// Environment variable names.
const (
	EnvWorkfDir = "work_dir"
)

// Environment names.
const (
	EnvDev        = "dev"
	EnvTest       = "test"
	EnvProduction = "prod"
)
