package anomaly

import "errors"

// ErrNotFound is returned when an anomaly id does not exist.
var ErrNotFound = errors.New("anomaly not found")
