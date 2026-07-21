// internal/auth/main_test.go
package auth

import (
	"fmt"
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
	fmt.Println("DEBUG: loading .env from:", envPath)
	err := godotenv.Load(envPath)
	fmt.Println("DEBUG: godotenv.Load error:", err)

	os.Exit(m.Run())
}