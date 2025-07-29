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
	assert := require.New(t)

	suite := &Suite{}

	assert.NoError(suite.Setup())

	t.Cleanup(suite.TearDown)

	// Run testdata tests
	testDir := "testdata"
	ents, err := os.ReadDir(testDir)
	assert.NoError(err)

	L, err := suite.lib.NewState()
	assert.NoError(err)

	L.OpenLibs()

	ltask.OpenLibs(L, suite.lib)

	t.Cleanup(L.Close)

	for _, ent := range ents {
		if ent.IsDir() {
			continue
		}
		if !strings.HasPrefix(ent.Name(), "test_") {
			continue
		}
		if !strings.HasSuffix(ent.Name(), ".lua") {
			continue
		}
		sname := strcase.ToCamel(strings.TrimPrefix(strings.TrimSuffix(ent.Name(), ".lua"), "test_"))
		scode, err := os.ReadFile(fmt.Sprintf("%s/%s", testDir, ent.Name()))
		assert.NoError(err)

		t.Run(sname, func(t *testing.T) {
			assert := require.New(t)

			assert.NoError(L.DoString(string(scode)))
		})
	}

	// Run tests in the suite
	tt := reflect.TypeOf(suite)
	for i := range tt.NumMethod() {
		method := tt.Method(i)
		if testFunc, ok := method.Func.Interface().(func(*Suite, *require.Assertions, *lua.State)); ok {
			t.Run(strings.TrimPrefix(method.Name, "Test"), func(t *testing.T) {
				assert := require.New(t)

				testFunc(suite, assert, L)
			})
		}
	}
}
