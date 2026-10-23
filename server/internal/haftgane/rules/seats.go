package rules

import "math/bits"

type Seat uint8

const NoSeat Seat = 255

type SeatSet uint8

func (s Seat) Valid(n uint8) bool { return n >= MinSeats && n <= MaxSeats && s < Seat(n) }
func (s Seat) Left(n uint8) Seat {
	if !s.Valid(n) {
		return NoSeat
	}
	return Seat((uint8(s) + 1) % n)
}
func (s Seat) Right(n uint8) Seat {
	if !s.Valid(n) {
		return NoSeat
	}
	return Seat((uint8(s) + n - 1) % n)
}
func (s Seat) Mask() SeatSet {
	if s < MaxSeats {
		return 1 << s
	}
	return 0
}
func (s SeatSet) Has(seat Seat) bool { return s&seat.Mask() != 0 }
func (s SeatSet) Count() int         { return bits.OnesCount8(uint8(s)) }
func AllSeats(n uint8) SeatSet {
	if n < MinSeats || n > MaxSeats {
		return 0
	}
	return (1 << n) - 1
}
