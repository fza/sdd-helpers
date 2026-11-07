package test

import (
	"context"
	"sync"
)

// Executables is a process table listing Paths for every prefix, or failing
// with Err.
type Executables struct {
	Paths []string
	Err   error
}

func (fake Executables) Executables(context.Context, string) ([]string, error) {
	return fake.Paths, fake.Err
}

// Starts records every process a test would have started, starting none.
type Starts struct {
	mutex   sync.Mutex
	started [][]string
}

func (fake *Starts) Start(_ context.Context, path string, args ...string) error {
	fake.mutex.Lock()
	defer fake.mutex.Unlock()

	fake.started = append(fake.started, append([]string{path}, args...))

	return nil
}

func (fake *Starts) Started() [][]string {
	fake.mutex.Lock()
	defer fake.mutex.Unlock()

	return fake.started
}
