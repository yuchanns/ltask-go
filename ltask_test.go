package ltask_test

import (
	"fmt"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.yuchanns.xyz/ltask"
	"go.yuchanns.xyz/lua"
)

type Suite struct {
}

func (s *Suite) Setup() (err error) {
	var path string
	switch runtime.GOOS {
	case "windows":
		path = "lua54.dll"
	case "linux":
		path = "liblua54.so"
	case "darwin":
		path = "liblua54.dylib"
	}
	err = lua.Init(fmt.Sprintf("../lua/lua54/.lua/lib/%s", path))
	if err != nil {
		return
	}

	return nil
}

func (s *Suite) TearDown() {
	_ = lua.Deinit()
}

func TestSuite(t *testing.T) {
	t.Parallel()
	assert := require.New(t)

	suite := &Suite{}

	assert.NoError(suite.Setup())

	t.Cleanup(suite.TearDown)

	t.Run("ltask", func(t *testing.T) {
		t.Parallel()
		assert := require.New(t)

		L := lua.NewState()

		L.OpenLibs()

		ltask.OpenLibs(L)

		t.Cleanup(L.Close)

		assert.NoError(L.DoFile("test.lua"))
	})

	tt := reflect.TypeOf(suite)
	for i := range tt.NumMethod() {
		method := tt.Method(i)
		if testFunc, ok := method.Func.Interface().(func(*Suite, *require.Assertions, *lua.State)); ok {
			t.Run(strings.TrimPrefix(method.Name, "Test"), func(t *testing.T) {
				t.Parallel()
				assert := require.New(t)
				L := lua.NewState()

				L.OpenLibs()

				t.Cleanup(L.Close)

				testFunc(suite, assert, L)
			})
		}
	}
}
