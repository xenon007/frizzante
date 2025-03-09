package frizzante

import (
	"database/sql"
	"fmt"
	"reflect"
	"strings"
)

func mysqlFindNextFallback(dest ...any) bool { return false }
func mysqlFindCloseFallback()                {}

type Mysql struct {
	database    *sql.DB
	recallError func(error)
}

// MysqlCreate creates a sql wrapper.
func MysqlCreate() *Mysql {
	return &Mysql{
		recallError: func(error) {},
	}
}

// SqlWithDatabase sets the sql database.
func SqlWithDatabase(self *Mysql, database *sql.DB) {
	self.database = database
}

// MysqlRecallError recalls errors notified by MysqlNotifyError.
func MysqlRecallError(self *Mysql, callback func(err error)) {
	self.recallError = callback
}

// MysqlNotifyError notifies an error.
//
// Recall errors with MysqlRecallError.
func MysqlNotifyError(self *Mysql, err error) {
	if nil == self.recallError {
		return
	}
	self.recallError(err)
}

// MysqlExecute executes sql queries that don't return rows, typically INSERT, UPDATE, DELETE queries.
func MysqlExecute(self *Mysql, query string, props ...any) *sql.Result {
	transaction, transactionError := self.database.Begin()
	if transactionError != nil {
		MysqlNotifyError(self, transactionError)
		return nil
	}

	statement, statementError := transaction.Prepare(query)
	if nil != statementError {
		MysqlNotifyError(self, statementError)
		return nil
	}

	result, execError := statement.Exec(props...)
	if execError != nil {
		MysqlNotifyError(self, execError)
		rollbackError := transaction.Rollback()
		if rollbackError != nil {
			MysqlNotifyError(self, rollbackError)
		}
		return nil
	}

	commitError := transaction.Commit()
	if commitError != nil {
		MysqlNotifyError(self, commitError)
		return nil
	}

	return &result
}

// MysqlFind executes a sql query that returns rows, typically a SELECT query.
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
func MysqlFind(self *Mysql, query string, props ...any) (next func(dest ...any) bool, close func()) {
	next = mysqlFindNextFallback
	close = mysqlFindCloseFallback

	statement, statementError := self.database.Prepare(query)
	if nil != statementError {
		MysqlNotifyError(self, statementError)
		return
	}
	defer statement.Close()

	rows, queryError := statement.Query(props...)
	if queryError != nil {
		MysqlNotifyError(self, queryError)
		return
	}

	next = func(dest ...any) bool {
		if !rows.Next() {
			return false
		}

		scanError := rows.Scan(dest...)
		if scanError != nil {
			MysqlNotifyError(self, scanError)
			return false
		}
		return true
	}
	close = func() {
		err := rows.Close()
		if err != nil {
			MysqlNotifyError(self, err)
		}
	}
	return
}

// MysqlCreateTable creates a table from a type.
func MysqlCreateTable[Table any](self *Mysql) {
	var query strings.Builder
	t := reflect.TypeFor[Table]()
	query.WriteString(fmt.Sprintf("create table `%s` (\n", t.Name()))
	count := t.NumField()
	for i := 0; i < count; i++ {
		field := t.Field(i)
		rules := field.Tag.Get("sql")
		if i > 0 {
			query.WriteString(",\n")
		}
		query.WriteString(fmt.Sprintf("`%s` %s", field.Name, rules))
	}
	query.WriteString("\n);")
	_, err := self.database.Exec(query.String())
	if err != nil {
		MysqlNotifyError(self, err)
	}
}
