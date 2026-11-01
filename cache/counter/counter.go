// Package counter supply counter service
package counter

// Fields define the counter's field and value
type Fields map[string]int64

// Counter service
type Counter interface {
	// GetName counter name
	GetName() string
	// Incr increase the counterID with fieldAndDelta
	Incr(counterID string, fieldAndDelta Fields) error

	// Get the fields of counterID
	Get(counterID string) (fields Fields, err error)

	// Del delete the counter whose id is `counterID``
	Del(counterID string) error
}

// Persist loads, saves, and deletes counter fields in persistent storage.
// Implementations must support concurrent calls. Operations are not serialized
// with Redis updates or background scans.
type Persist interface {
	// Load the fields of counterID from persist storage
	Load(counterID string) (fields Fields, err error)

	// Del delete the counter whose id is `counterID``
	Del(counterID string) (deleted bool, err error)

	// Store save the value of fields with counterID
	Store(counterID string, fields Fields) error
}
