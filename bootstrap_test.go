package ltask_test

func (s *Suite) TestBootstrapInit(testWithFile func(file string)) {
	testWithFile("bootstrap.lua")
}
