package frizzante

import (
	"errors"
	"testing"
)

func TestSqlCreate(test *testing.T) {
	sql := MysqlCreate()
	if nil == sql {
		test.Fatal("could not create sql")
	}
}

func TestSqlWithErrorReceiver(test *testing.T) {
	sql := MysqlCreate()
	var receivedError error
	MysqlRecallError(sql, func(err error) {
		receivedError = err
	})
	MysqlNotifyError(sql, errors.New("test error"))
	if "test error" != receivedError.Error() {
		test.Fatal("could not notify sql error")
	}
}
