package redirect

import "net/http"

// Engine of all redirection.
type Engine interface {
	http.Handler
	Reload() error // reload configuration from storage
}

// Stats consumer.
type StatWriter interface {
	Touch(url string) // Touch resource and increment counter (hot operation, should be fast)
}

// Stats reader.
type StatReader interface {
	Visits(url string) int64 // Get number of visits for specific service/url
}

// Stats reader and writer.
type Stats interface {
	StatWriter
	StatReader
}

// Single rule for redirection.
type Rule struct {
	URL              string // Matching URL (aka service name)
	LocationTemplate string // Go-Template of target location
}

// Rules storage type.
type Storage interface {
	Set(url string, locationTemplate string) error // add or replace rule
	Get(url string) (string, bool)                 // get location template. should return true if exists
	Remove(url string) error                       // remove rule (or ignore if not exists)
	All() ([]*Rule, error)                         // dump all save rules
	Reload() error                                 // reload storage and fill the internal cache
}
