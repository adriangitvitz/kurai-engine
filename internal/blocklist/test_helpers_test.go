package blocklist

import (
	"os"
	"testing"
)

func openIfExists(t *testing.T, path string) (*os.File, error) {
	t.Helper()
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		t.Fatal(err)
	}
	return f, nil
}
