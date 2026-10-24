package tableroom

// SweepSeats is the seat counts a seat-count sweep plays: every count from
// MinSeats to MaxSeats, or, in the race build, the two ends only. The race
// detector looks for unsynchronized memory access, which the same
// goroutines and code paths show at three seats and at seven; the counts
// between add game size, not concurrency, and cost most of the race
// build's time. Every count still runs without the race detector, in
// verify-quick and CI's go job.
func SweepSeats() []int {
	if RaceEnabled {
		return []int{MinSeats, MaxSeats}
	}
	seats := make([]int, 0, MaxSeats-MinSeats+1)
	for n := MinSeats; n <= MaxSeats; n++ {
		seats = append(seats, n)
	}
	return seats
}
