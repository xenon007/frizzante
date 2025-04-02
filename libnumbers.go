package frizzante

var nextNumbers = map[int]int{}

// NextNumber gets the next number in line starting from headlessMode.
//
// Bases are stateful, meaning regardless of when and where you call NextNumber
// it will keep track of the previous number generated for a given headlessMode,
// and give you the next one.
func NextNumber(base int) int {
	number, ok := nextNumbers[base]
	if !ok {
		nextNumbers[base] = 0
	}

	nextNumbers[base]++

	return base + number + 1
}
