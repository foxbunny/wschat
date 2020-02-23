package command_socket

type RadioParams struct {
	frequency       float64
	spreadingFactor int
	bandwidth       int
	codingRate      int
}

const DEFAULT_FREQUENCY = 1000.0
const DEFAULT_SPREADING_FACTOR = 12
const DEFAULT_BANDWIDTH = 400
const DEFAULT_CODING_RATE = 5

var Bandwidths = map[int]int{
	200:  52,
	400:  38,
	800:  24,
	1600: 10,
}

var SpreadingFactors = map[int]int{
	5:  80,
	6:  96,
	7:  112,
	8:  128,
	9:  144,
	10: 160,
	11: 176,
	12: 192,
}

var CodingRates = map[int]int{
	5: 1,
	6: 2,
	7: 3,
	8: 4,
}
