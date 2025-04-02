package frizzante

import (
	"errors"
	"testing"
)

func TestSqlOperatorCreate(test *testing.T) {
	sql := SqlOperatorCreate()
	if nil == sql {
		test.Fatal("could not create sql")
	}
}

func TestSqlOperatorNotifier(test *testing.T) {
	sql := SqlOperatorCreate()
	notifier := NotifierCreate()
	SqlOperatorWithNotifier(sql, notifier)
	var receivedError error
	NotifierReceiveError(sql.notifier, func(err error) {
		receivedError = err
	})
	NotifierSendError(sql.notifier, errors.New("test error"))
	if "test error" != receivedError.Error() {
		test.Fatal("could not notify sql error")
	}
}
