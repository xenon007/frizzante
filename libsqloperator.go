package frizzante

import (
	"database/sql"
)

func sqlOperatorFindNextFallback(dest ...any) bool { return false }
func sqlOperatorFindCloseFallback()                {}

type SqlOperator struct {
	database *sql.DB
	notifier *Notifier
}

func SqlOperatorCreate() *SqlOperator {
	return &SqlOperator{
		notifier: NotifierCreate(),
	}
}

// SqlOperatorWithNotifier sets the sql notifier.
func SqlOperatorWithNotifier(self *SqlOperator, notifier *Notifier) {
	self.notifier = notifier
}

// SqlOperatorExecute executes sql queries that don't return rows, typically INSERT, UPDATE, DELETE queries.
func SqlOperatorExecute(self *SqlOperator, query string, props ...any) *sql.Result {
	transaction, transactionError := self.database.Begin()
	if transactionError != nil {
		NotifierSendError(self.notifier, transactionError)
		return nil
	}

	result, execError := transaction.Exec(query, props...)
	if execError != nil {
		NotifierSendError(self.notifier, execError)
		rollbackError := transaction.Rollback()
		if rollbackError != nil {
			NotifierSendError(self.notifier, rollbackError)
		}
		return nil
	}

	commitError := transaction.Commit()
	if commitError != nil {
		NotifierSendError(self.notifier, commitError)
		return nil
	}

	return &result
}

// SqlOperatorFind executes a sql query that returns rows, typically a SELECT query.
//
// It returns a next function and a close function.
//
// Use next to project the next row onto dest.
//
// Next will return false if where are no more rows available.
//
// Use close to close the database context and prevent any subsequent enumerations.
//
// Whenever next returns false, the database context is closed automatically as if calling close.
func SqlOperatorFind(self *SqlOperator, query string, props ...any) (next func(dest ...any) bool, close func()) {
	next = sqlOperatorFindNextFallback
	close = sqlOperatorFindCloseFallback

	rows, queryError := self.database.Query(query, props...)
	if queryError != nil {
		NotifierSendError(self.notifier, queryError)
		return
	}

	next = func(dest ...any) bool {
		if !rows.Next() {
			return false
		}

		err := rows.Scan(dest...)
		if err != nil {
			return false
		}
		return true
	}
	close = func() {
		err := rows.Close()
		if err != nil {
			NotifierSendError(self.notifier, err)
		}
	}
	return
}
