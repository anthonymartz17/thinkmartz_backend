// internal/auth/main_test.go
package auth

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/joho/godotenv"
)

func TestMain(m *testing.M) {

	_, thisFile, _, _ := runtime.Caller(0)
	repoRoot := filepath.Join(filepath.Dir(thisFile), "..", "..")
	envPath := filepath.Join(repoRoot, ".env")
	_ = godotenv.Load(envPath)

	os.Exit(m.Run())
}
