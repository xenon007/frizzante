package frizzante

import (
	"database/sql"
	"fmt"
	"reflect"
	"strings"
)

func mysqlFindNextFallback(dest ...any) bool { return false }
func mysqlFindCloseFallback()                {}

type SqlDialect int64

const (
	SqlDialectMysql      SqlDialect = 0
	SqlDialectPostgresql SqlDialect = 1
)

type Sql struct {
	database    *sql.DB
	recallError func(error)
	dialect     SqlDialect
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

// SqlWithDialect sets the sql dialect.
func SqlWithDialect(self *Sql, dialect SqlDialect) {
	self.dialect = dialect
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

	statement, statementError := transaction.Prepare(query)
	if nil != statementError {
		SqlNotifyError(self, statementError)
		return nil
	}

	result, execError := statement.Exec(props...)
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
	next = mysqlFindNextFallback
	close = mysqlFindCloseFallback

	statement, statementError := self.database.Prepare(query)
	if nil != statementError {
		SqlNotifyError(self, statementError)
		return
	}
	defer statement.Close()

	rows, queryError := statement.Query(props...)
	if queryError != nil {
		SqlNotifyError(self, queryError)
		return
	}

	next = func(dest ...any) bool {
		if !rows.Next() {
			return false
		}

		scanError := rows.Scan(dest...)
		if scanError != nil {
			SqlNotifyError(self, scanError)
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

// SqlDropTable drops a table from a type if it exists.
func SqlDropTable[Table any](self *Sql) {
	var query strings.Builder
	t := reflect.TypeFor[Table]()
	switch self.dialect {
	case SqlDialectMysql:
		query.WriteString(fmt.Sprintf("drop table if not exists `%s`;", t.Name()))
	case SqlDialectPostgresql:
		query.WriteString(fmt.Sprintf("drop table if not exists \"%s\";", t.Name()))
	default:
		query.WriteString(fmt.Sprintf("drop table if not exists `%s`;", t.Name()))
	}
	_, err := self.database.Exec(query.String())
	if err != nil {
		SqlNotifyError(self, err)
	}
}

// SqlCreateTable creates a table from a type if it doesn't already exist.
func SqlCreateTable[Table any](self *Sql) {
	var query strings.Builder
	var meta strings.Builder
	t := reflect.TypeFor[Table]()
	structName := t.Name()
	expectedIdentifier := structName + "Id"
	hasIdentifier := false
	switch self.dialect {
	case SqlDialectMysql:
		query.WriteString(fmt.Sprintf("create table if not exists `%s` ", structName))
	case SqlDialectPostgresql:
		query.WriteString(fmt.Sprintf("create table if not exists \"%s\" ", structName))
	default:
		query.WriteString(fmt.Sprintf("create table if not exists `%s` ", structName))
	}
	query.WriteString("(\n")
	count := t.NumField()
	for i := 0; i < count; i++ {
		if i > 0 {
			query.WriteString(",\n")
		}
		field := t.Field(i)
		fieldName := field.Name
		fieldType := field.Type
		fieldTypeName := fieldType.Name()
		fieldCount := fieldType.NumField()
		fieldIsComplex := fieldCount > 0

		if !fieldIsComplex {
			if expectedIdentifier == fieldName {
				hasIdentifier = true
			}
			rules := field.Tag.Get("sql")

			switch self.dialect {
			case SqlDialectMysql:
				query.WriteString(fmt.Sprintf("`%s` %s", fieldName, rules))
			case SqlDialectPostgresql:
				query.WriteString(fmt.Sprintf("\"%s\" %s", fieldName, rules))
			default:
				query.WriteString(fmt.Sprintf("`%s` %s", fieldName, rules))
			}

			continue
		}

		expectedForeignIdentifier := fieldName + "Id"

		for j := 0; j < fieldCount; j++ {
			fieldOfField := fieldType.Field(j)
			fieldOfFieldName := fieldOfField.Name
			if expectedForeignIdentifier == fieldOfFieldName {

				switch self.dialect {
				case SqlDialectMysql:
					meta.WriteString(fmt.Sprintf("foreign key (`%s`) references `%s` (`%s`)", fieldOfFieldName, structName, fieldName))
				case SqlDialectPostgresql:
					meta.WriteString(fmt.Sprintf("foreign key (\"%s\") references \"%s\" (\"%s\")", fieldOfFieldName, structName, fieldName))
				default:
					meta.WriteString(fmt.Sprintf("foreign key (`%s`) references `%s` (`%s`)", fieldOfFieldName, structName, fieldName))
				}

				break
			}
		}
		SqlNotifyError(self,
			fmt.Errorf(
				"field `%s` is a `%s` structure and is expected to have a `%s` identifier field, but it doesn't",
				fieldName,
				fieldTypeName,
				expectedForeignIdentifier,
			),
		)
	}

	if !hasIdentifier {
		SqlNotifyError(self,
			fmt.Errorf(
				"structure `%s` is expected to have a `%s` identifier field, but it doesn't",
				structName,
				expectedIdentifier,
			),
		)
		return
	}

	switch self.dialect {
	case SqlDialectMysql:
		meta.WriteString(fmt.Sprintf("primary key (`%s`)", expectedIdentifier))
	case SqlDialectPostgresql:
		meta.WriteString(fmt.Sprintf("primary key (\"%s\")", expectedIdentifier))
	default:
		meta.WriteString(fmt.Sprintf("primary key (`%s`)", expectedIdentifier))
	}

	query.WriteString(",\n")
	query.WriteString(meta.String())

	query.WriteString("\n);")
	_, err := self.database.Exec(query.String())
	if err != nil {
		SqlNotifyError(self, err)
	}
}
