package app_test

import (
	"io"
	"log"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	if os.Getenv("TEST_VERBOSE_LOGS") == "" {
		log.SetOutput(io.Discard)
	}
	os.Exit(m.Run())
}
