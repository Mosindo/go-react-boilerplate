package main

import (
	"io"
	"log"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestMain(m *testing.M) {
	log.SetOutput(io.Discard)
	gin.DefaultWriter = io.Discard
	os.Exit(m.Run())
}
