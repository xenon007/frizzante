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
		dialect:     SqlDialectMysql,
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
		query.WriteString(fmt.Sprintf("drop table if exists `%s`;", t.Name()))
	case SqlDialectPostgresql:
		query.WriteString(fmt.Sprintf("drop table if exists \"%s\";", t.Name()))
	}
	_, err := self.database.Exec(query.String())
	if err != nil {
		SqlNotifyError(self, err)
	}
}

// SqlCreateTable creates a table from a type if it doesn't already exist.
func SqlCreateTable[Table any](self *Sql) {
	var query strings.Builder
	t := reflect.TypeFor[Table]()
	structName := t.Name()
	expectedId := "Id"
	hasId := false
	switch self.dialect {
	case SqlDialectMysql:
		query.WriteString(fmt.Sprintf("create table if not exists `%s` ", structName))
	case SqlDialectPostgresql:
		query.WriteString(fmt.Sprintf("create table if not exists \"%s\" ", structName))
	}
	query.WriteString("(\n")
	count := t.NumField()
	for i := 0; i < count; i++ {
		field := t.Field(i)
		fieldName := field.Name
		fieldType := field.Type
		fieldTypeName := fieldType.Name()
		fieldCount := 0
		fieldIsComplex := false

		if "" == fieldTypeName {
			SqlNotifyError(self, fmt.Errorf("failed to create table `%s` because field `%s` is a pointer, which is not allowed", structName, fieldName))
			return
		}

		if "string" != fieldTypeName &&
			"int" != fieldTypeName &&
			"int8" != fieldTypeName &&
			"int16" != fieldTypeName &&
			"int32" != fieldTypeName &&
			"int64" != fieldTypeName &&
			"float32" != fieldTypeName &&
			"float64" != fieldTypeName &&
			"bool" != fieldTypeName {
			fieldCount = fieldType.NumField()
			fieldIsComplex = true
		}

		if !fieldIsComplex {
			if expectedId == fieldName {
				hasId = true
			}
			rules := field.Tag.Get("sql")

			switch self.dialect {
			case SqlDialectMysql:
				query.WriteString(fmt.Sprintf("`%s` %s", fieldName, rules))
			case SqlDialectPostgresql:
				query.WriteString(fmt.Sprintf("\"%s\" %s", fieldName, rules))
			}

			if i < count {
				query.WriteString(",\n")
			}
			continue
		}

		expectedForeignId := "Id"
		hasForeignId := false
		for j := 0; j < fieldCount; j++ {
			fieldForeign := fieldType.Field(j)
			if expectedForeignId == fieldForeign.Name {
				rules := fieldForeign.Tag.Get("sql")
				switch self.dialect {
				case SqlDialectMysql:
					query.WriteString(fmt.Sprintf("`%s` %s", fieldName, rules))
				case SqlDialectPostgresql:
					query.WriteString(fmt.Sprintf("\"%s\" %s", fieldName, rules))
				}

				query.WriteString(",\n")

				switch self.dialect {
				case SqlDialectMysql:
					query.WriteString(fmt.Sprintf("foreign key (`%s`) references `%s` (`%s`)", fieldName, field.Type.Name(), expectedForeignId))
				case SqlDialectPostgresql:
					query.WriteString(fmt.Sprintf("foreign key (\"%s\") references \"%s\" (\"%s\")", fieldName, field.Type.Name(), expectedForeignId))
				}
				hasForeignId = true
				break
			}
		}
		if !hasForeignId {
			SqlNotifyError(self,
				fmt.Errorf(
					"field `%s` of type `%s` is expected to have an identifier field named `%s`, but it doesn't",
					fieldName,
					fieldTypeName,
					expectedForeignId,
				),
			)
		}

		if i < count {
			query.WriteString(",\n")
		}
	}

	if !hasId {
		SqlNotifyError(self,
			fmt.Errorf(
				"structure `%s` is expected to have an identifier field named `%s`, but it doesn't",
				structName,
				expectedId,
			),
		)
		return
	}

	switch self.dialect {
	case SqlDialectMysql:
		query.WriteString(fmt.Sprintf("primary key (`%s`)", expectedId))
	case SqlDialectPostgresql:
		query.WriteString(fmt.Sprintf("primary key (\"%s\")", expectedId))
	}

	query.WriteString("\n);")
	_, err := self.database.Exec(query.String())
	if err != nil {
		SqlNotifyError(self, err)
	}
}
