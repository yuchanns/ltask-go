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
	lib *lua.Lib
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
	s.lib, err = lua.New(fmt.Sprintf("../lua/lua54/.lua/lib/%s", path))
	if err != nil {
		return
	}

	return nil
}

func (s *Suite) TearDown() {
	if s.lib == nil {
		return
	}
	s.lib.Close()
}

func TestSuite(t *testing.T) {
	t.Parallel()

	assert := require.New(t)

	suite := &Suite{}

	assert.NoError(suite.Setup())

	t.Cleanup(suite.TearDown)

	tt := reflect.TypeOf(suite)
	for i := range tt.NumMethod() {
		method := tt.Method(i)
		testFunc, ok := method.Func.Interface().(func(*Suite, func(string)))
		if !ok {
			continue
		}
		t.Run(strings.TrimPrefix(method.Name, "Test"), func(t *testing.T) {
			t.Parallel()
			assert := require.New(t)

			L, err := suite.lib.NewState()
			assert.NoError(err)

			L.OpenLibs()

			t.Cleanup(L.Close)

			ltask.OpenLibs(L)

			testFunc(suite, func(file string) {
				assert.NoError(L.DoFile(fmt.Sprintf("testdata/%s", file)))
			})
		})
	}
}
