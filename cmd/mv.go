package cmd

import (
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/spf13/cobra"
)

var (
	runs int
	size int
)

func init() {
	rootCmd.AddCommand(mvCmd)
	mvCmd.Flags().IntVarP(&runs, "runs", "r", 1000, "number of times to run the matrix vector multiplication")
	mvCmd.Flags().IntVarP(&size, "size", "s", 1000, "size of the square matrix")
}

var mvCmd = &cobra.Command{
	Use:   "mv",
	Short: "Calculate Matrix Vector",
	Args:  cobra.MaximumNArgs(0),
	Run: func(cmd *cobra.Command, args []string) {

		// Initialize variables for time measurement and performance metrics
		var startTime, stopTime time.Time
		var diffTime time.Duration
		var seconds, nanoseconds, runtime, flops, mflops float32

		// Create and initialize the matrix, vector, and result arrays
		matrix := make([][]int, size)
		for i := range matrix {
			matrix[i] = make([]int, size)
		}
		vector := make([]int, size)
		result := make([]int, size)

		// Populate the matrix and vector with values
		for i := range size {
			for j := range size {
				matrix[i][j] = i*size + j + 1
			}
			vector[i] = i + 1
			result[i] = 0
		}

		// Create a wait group to synchronize goroutines
		var wg sync.WaitGroup
		wg.Add(runs)

		// Start the timer and launch goroutines to perform matrix vector multiplication concurrently
		startTime = time.Now()
		for range runs {
			go func() {
				defer wg.Done()

				// Create a local result array for each goroutine
				localResult := make([]int, size)

				// Perform the matrix vector multiplication
				for i := range size {
					for j := range size {
						localResult[i] += matrix[i][j] * vector[j]
					}
				}

				// Update the global result array using the local result
				for i := range size {
					result[i] += localResult[i]
				}
			}()
		}

		// Wait for all goroutines to finish
		wg.Wait()

		// Stop the timer and calculate the runtime
		stopTime = time.Now()
		diffTime = stopTime.Sub(startTime)
		seconds = float32(diffTime.Seconds())
		nanoseconds = float32(diffTime.Nanoseconds())
		runtime = seconds + nanoseconds/1000000000

		// Calculate the number of floating-point operations (FLOPS) and megaFLOPS (MFLOPS)
		flops = float32((2 * size * size)) / runtime * float32(runs)
		mflops = flops / 1000000

		// Print the results
		fmt.Println("Median Runtime:", runtime/float32(runs))
		fmt.Println("Runtime:     ", round(float64(seconds), 2))
		fmt.Println("Runs:        ", runs)
		fmt.Println("MFlops:      ", mflops)
	},
}

// Function to round a number to a specified precision.
func round(num float64, precision int) float64 {
	output := math.Pow(10, float64(precision))

	return float64(int((num*output)+math.Copysign(0.5, (num*output)))) / output
}
