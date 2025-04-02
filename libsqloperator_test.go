package frizzante

import (
	"testing"
)

func TestSqlOperatorCreate(test *testing.T) {
	sql := SqlOperatorCreate()
	if nil == sql {
		test.Fatal("could not create sql")
	}
}
