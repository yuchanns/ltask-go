package ltask_test

import (
	"fmt"
	"os"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/iancoleman/strcase"
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

	// Run testdata tests
	testDir := "testdata"
	ents, err := os.ReadDir(testDir)
	assert.NoError(err)

	for _, ent := range ents {
		if ent.IsDir() {
			continue
		}
		sname := ent.Name()
		if !strings.HasPrefix(sname, "test_") {
			continue
		}
		testName := strings.TrimPrefix(sname, "test_")
		if !strings.HasSuffix(testName, ".lua") {
			continue
		}
		testName = strings.TrimSuffix(testName, ".lua")

		t.Run(strcase.ToCamel(testName), func(t *testing.T) {
			t.Parallel()
			assert := require.New(t)

			L, err := suite.lib.NewState()
			assert.NoError(err)

			L.OpenLibs()

			ltask.OpenLibs(L, suite.lib)

			t.Cleanup(L.Close)

			assert.NoError(L.DoFile(fmt.Sprintf("%s/%s", testDir, sname)))
		})
	}

	// Run tests in the suite
	tt := reflect.TypeOf(suite)
	for i := range tt.NumMethod() {
		method := tt.Method(i)
		if testFunc, ok := method.Func.Interface().(func(*Suite, *require.Assertions, *lua.State)); ok {
			t.Run(strings.TrimPrefix(method.Name, "Test"), func(t *testing.T) {
				t.Parallel()
				assert := require.New(t)
				L, err := suite.lib.NewState()
				assert.NoError(err)

				L.OpenLibs()

				t.Cleanup(L.Close)

				testFunc(suite, assert, L)
			})
		}
	}
}
