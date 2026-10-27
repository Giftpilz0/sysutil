package bench

import (
	"context"
	"testing"
)

func TestRun(t *testing.T) {
	var last int

	res, err := Run(context.Background(), Config{Runs: 10, Size: 3, Workers: 4}, func(done int) {
		last = max(last, done)
	})
	if err != nil {
		t.Fatal(err)
	}

	// Last row of [[1 2 3] [4 5 6] [7 8 9]] · [1 2 3] is 7+16+27 = 50.
	if res.Checksum != 500 {
		t.Errorf("checksum = %v, want 500", res.Checksum)
	}

	if last != 10 {
		t.Errorf("progress reported %d runs, want 10", last)
	}
}

func TestRunInvalid(t *testing.T) {
	if _, err := Run(context.Background(), Config{Runs: 0, Size: 10}, nil); err == nil {
		t.Fatal("expected error")
	}
}
