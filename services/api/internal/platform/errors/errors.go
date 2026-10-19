package platformerrors

import (
	"fmt"

	"github.com/gin-gonic/gin"
)

func Wrap(operation string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", operation, err)
}

// Respond writes the standard error envelope: a human readable message plus a
// stable machine-readable code that clients can switch on.
func Respond(c *gin.Context, status int, code, message string) {
	c.JSON(status, gin.H{"error": message, "code": code})
}
