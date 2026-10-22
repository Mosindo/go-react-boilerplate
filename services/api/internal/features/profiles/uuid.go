package profiles

import "regexp"

var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// IsUUID reports whether v is a canonical UUID string. Handlers use it to turn
// malformed path parameters into 404/400 instead of database errors.
func IsUUID(v string) bool { return uuidPattern.MatchString(v) }
