package frizzante

import (
	"errors"
	"testing"
)

func TestSqlCreate(test *testing.T) {
	sql := SqlCreate()
	if nil == sql {
		test.Fatal("could not create sql")
	}
}

func TestSqlWithErrorReceiver(test *testing.T) {
	sql := SqlCreate()
	var receivedError error
	SqlWithErrorReceiver(sql, func(err error) {
		receivedError = err
	})
	SqlNotifyError(sql, errors.New("test error"))
	if "test error" != receivedError.Error() {
		test.Fatal("could not notify sql error")
	}
}
