package worker

import "testing"

func TestPool(t *testing.T) {

	pool := NewPool(1)
	pool.Start()

	var count int
	_ = pool.Enqueue(func() error {
		count++
		return nil
	})

	pool.Stop()
	_ = <-pool.Complete()

	if count != 1 {
		t.Errorf("Expected a count of 1, got %v\n", count)
	}

}
