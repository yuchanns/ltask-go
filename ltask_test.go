package ltask_test

import (
	"testing"

	"go.yuchanns.xyz/ltask"
)

func TestHello(t *testing.T) {
	err := ltask.Hello()
	if err != nil {
		t.Errorf("Failed to execute Lua script: %v", err)
	}
}
