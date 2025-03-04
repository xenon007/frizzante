package frizzante

import (
	"database/sql"
	"fmt"
	"reflect"
	"strings"
)

func sqlFindNextFallback(dest ...any) bool { return false }
func sqlFindCloseFallback()                {}

type Sql struct {
	database    *sql.DB
	recallError func(error)
}

// SqlCreate creates a sql wrapper.
func SqlCreate() *Sql {
	return &Sql{
		recallError: func(error) {},
	}
}

// SqlWithDatabase sets the sql database.
func SqlWithDatabase(self *Sql, database *sql.DB) {
	self.database = database
}

// SqlRecallError recalls errors notified by SqlNotifyError.
func SqlRecallError(self *Sql, callback func(err error)) {
	self.recallError = callback
}

// SqlNotifyError notifies an error.
//
// Recall errors with SqlRecallError.
func SqlNotifyError(self *Sql, err error) {
	if nil == self.recallError {
		return
	}
	self.recallError(err)
}

// SqlExecute executes sql queries that don't return rows, typically INSERT, UPDATE, DELETE queries.
func SqlExecute(self *Sql, query string, props ...any) *sql.Result {
	transaction, transactionError := self.database.Begin()
	if transactionError != nil {
		SqlNotifyError(self, transactionError)
		return nil
	}

	result, execError := transaction.Exec(query, props...)
	if execError != nil {
		SqlNotifyError(self, execError)
		rollbackError := transaction.Rollback()
		if rollbackError != nil {
			SqlNotifyError(self, rollbackError)
		}
		return nil
	}

	commitError := transaction.Commit()
	if commitError != nil {
		SqlNotifyError(self, commitError)
		return nil
	}

	return &result
}

// SqlFind executes a sql query that returns rows, typically a SELECT query.
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
func SqlFind(self *Sql, query string, props ...any) (next func(dest ...any) bool, close func()) {
	next = sqlFindNextFallback
	close = sqlFindCloseFallback

	rows, queryError := self.database.Query(query, props...)
	if queryError != nil {
		SqlNotifyError(self, queryError)
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
			SqlNotifyError(self, err)
		}
	}
	return
}

// SqlCreateTable creates a table from a type.
func SqlCreateTable[Table any](self *Sql) {
	var query strings.Builder
	t := reflect.TypeFor[Table]()
	query.WriteString(fmt.Sprintf("create table `%s`(\n", t.Name()))
	count := t.NumField()
	for i := 0; i < count; i++ {
		field := t.Field(i)
		rules := field.Tag.Get("sql")
		if i > 0 {
			query.WriteString(",\n")
		}
		query.WriteString(fmt.Sprintf("`%s` %s", field.Name, rules))
	}
	query.WriteString(");")
	_, err := self.database.Exec(query.String())
	if err != nil {
		SqlNotifyError(self, err)
	}
}
