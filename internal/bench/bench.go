package bench

import (
	"context"
	"errors"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

type Config struct {
	Runs    int
	Size    int
	Workers int // 0 means runtime.NumCPU()
}

type Result struct {
	Config
	Elapsed  time.Duration
	PerRun   time.Duration
	MFLOPS   float64
	Checksum float64
}

func DefaultConfig() Config {
	return Config{Runs: 1000, Size: 1000, Workers: runtime.NumCPU()}
}

// progress may be nil and is called concurrently from the workers.
func Run(ctx context.Context, cfg Config, progress func(done int)) (Result, error) {
	if cfg.Runs <= 0 || cfg.Size <= 0 {
		return Result{}, errors.New("runs and size must be positive")
	}

	if cfg.Workers <= 0 {
		cfg.Workers = runtime.NumCPU()
	}
	cfg.Workers = min(cfg.Workers, cfg.Runs)

	n := cfg.Size
	matrix := make([]float64, n*n)
	vector := make([]float64, n)
	for i := range n {
		for j := range n {
			matrix[i*n+j] = float64(i*n + j + 1)
		}
		vector[i] = float64(i + 1)
	}

	var (
		next     atomic.Int64
		done     atomic.Int64
		wg       sync.WaitGroup
		mu       sync.Mutex
		checksum float64
	)

	start := time.Now()

	for range cfg.Workers {
		wg.Go(func() {
			result := make([]float64, n)
			var local float64

			for next.Add(1) <= int64(cfg.Runs) {
				if ctx.Err() != nil {
					return
				}

				for i := range n {
					row := matrix[i*n : (i+1)*n]
					var sum float64
					for j, v := range row {
						sum += v * vector[j]
					}
					result[i] = sum
				}
				local += result[n-1]

				d := done.Add(1)
				if progress != nil {
					progress(int(d))
				}
			}

			mu.Lock()
			checksum += local
			mu.Unlock()
		})
	}

	wg.Wait()
	elapsed := time.Since(start)

	if err := ctx.Err(); err != nil {
		return Result{}, err
	}

	ops := 2 * float64(n) * float64(n) * float64(cfg.Runs)

	return Result{
		Config:   cfg,
		Elapsed:  elapsed,
		PerRun:   elapsed / time.Duration(cfg.Runs),
		MFLOPS:   ops / elapsed.Seconds() / 1e6,
		Checksum: checksum,
	}, nil
}
