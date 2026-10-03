// Package validate holds small input validators shared across features.
package validate

import "regexp"

var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// IsUUID reports whether s is a canonical textual UUID. Handlers use it before
// handing path parameters to PostgreSQL, which would otherwise answer with a 500.
func IsUUID(s string) bool { return uuidPattern.MatchString(s) }
