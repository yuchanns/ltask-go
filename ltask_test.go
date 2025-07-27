package ltask_test

import (
	"fmt"
	"os"
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

	testDir := "testdata"
	ents, err := os.ReadDir(testDir)
	assert.NoError(err)

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
			t.Parallel()
			assert := require.New(t)

			L, err := suite.lib.NewState()
			assert.NoError(err)

			L.OpenLibs()

			ltask.OpenLibs(L, suite.lib)

			t.Cleanup(L.Close)

			assert.NoError(L.DoString(string(scode)))
		})
	}
}
