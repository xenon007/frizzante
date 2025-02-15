package frizzante

import (
	"fmt"
	"testing"
)

func TestNextNumber(test *testing.T) {
	number := NextNumber(7)
	if 8 != number {
		test.Fatal(fmt.Sprintf("Expected next number to be 8, received %d instead", number))
	}

	number = NextNumber(7)
	if 9 != number {
		test.Fatal(fmt.Sprintf("Expected next number to be 9, received %d instead", number))
	}

	number = NextNumber(7)
	if 10 != number {
		test.Fatal(fmt.Sprintf("Expected next number to be 10, received %d instead", number))
	}

	number = NextNumber(8)
	if 9 != number {
		test.Fatal(fmt.Sprintf("Expected next number to be 9, received %d instead", number))
	}

	number = NextNumber(8)
	if 10 != number {
		test.Fatal(fmt.Sprintf("Expected next number to be 10, received %d instead", number))
	}

	number = NextNumber(8)
	if 11 != number {
		test.Fatal(fmt.Sprintf("Expected next number to be 11, received %d instead", number))
	}
}
