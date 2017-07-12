package services

import (
	"testing"
)

func TestLocks(t *testing.T) {
	service := ServiceConfig{
		Name:       "myservice",
		ConfigFile: "path/to/config",
	}

	var err error

	withLock, done, err := service.ObtainLock("testing")
	if err != nil {
		t.Fatal(err)
	}

	err = service.doStop(OperationConfig{}, ContextOverride{}, nil)
	if err == nil || err.Error() != "service locked: testing" {
		t.Errorf("Error was not as expected when locked: %v", err)
	}

	err = withLock.doStop(OperationConfig{}, ContextOverride{}, nil)
	if err != nil {
		t.Error(err)
	}

	err = done()
	if err != nil {
		t.Error(err)
	}

	err = service.doStop(OperationConfig{}, ContextOverride{}, nil)
	if err != nil {
		t.Error(err)
	}
}
