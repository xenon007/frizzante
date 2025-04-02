package frizzante

import "database/sql"

func sqlFindNextFallback(dest ...any) bool { return false }
func sqlFindCloseFallback()                {}

type SqlDialect int64

const (
	SqlDialectMysql      SqlDialect = 0
	SqlDialectPostgresql SqlDialect = 1
)

type Sql struct {
	database *sql.DB
	dialect  SqlDialect
	notifier *Notifier
}

// SqlCreate creates a sql wrapper.
func SqlCreate() *Sql {
	return &Sql{
		dialect: SqlDialectMysql,
	}
}

// SqlWithNotifier sets the sql notifier.
func SqlWithNotifier(self *Sql, notifier *Notifier) {
	self.notifier = notifier
}

// SqlWithDatabase sets the sql database.
func SqlWithDatabase(self *Sql, database *sql.DB) {
	self.database = database
}

// SqlWithDialect sets the sql dialect.
func SqlWithDialect(self *Sql, dialect SqlDialect) {
	self.dialect = dialect
}

// SqlExecute executes sql queries that don't return rows, typically INSERT, UPDATE, DELETE queries.
func SqlExecute(self *Sql, query string, props ...any) *sql.Result {
	transaction, transactionError := self.database.Begin()
	if transactionError != nil {
		NotifierSendError(self.notifier, transactionError)
		NotifierSendError(self.notifier, transactionError)
		return nil
	}

	statement, statementError := transaction.Prepare(query)
	if nil != statementError {
		NotifierSendError(self.notifier, statementError)
		return nil
	}

	result, execError := statement.Exec(props...)
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

	statement, statementError := self.database.Prepare(query)
	if nil != statementError {
		NotifierSendError(self.notifier, statementError)
		return
	}
	defer statement.Close()

	rows, queryError := statement.Query(props...)
	if queryError != nil {
		NotifierSendError(self.notifier, queryError)
		return
	}

	next = func(dest ...any) bool {
		if !rows.Next() {
			return false
		}

		scanError := rows.Scan(dest...)
		if scanError != nil {
			NotifierSendError(self.notifier, scanError)
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
