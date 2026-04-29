package tests

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	if os.Getenv("WP2EMDASH_E2E_TEST_ENABLED") != "true" {
		os.Exit(0)
	}
	os.Exit(m.Run())
}
