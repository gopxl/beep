package testtools

import (
	"path/filepath"
	"runtime"
)

var (
	// See https://stackoverflow.com/a/38644571
	_, callerPath, _, _ = runtime.Caller(0)
	TestDataDir         = filepath.Join(filepath.Dir(callerPath), "../testdata")
)

func TestFilePath(path string) string {
	return filepath.Join(TestDataDir, path)
}
