package frizzante

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
)

type Schema struct {
	sql      *SqlOperator
	notifier *Notifier
}

// SchemaCreate creates a new schema.
func SchemaCreate() *Schema {
	return &Schema{
		notifier: NotifierCreate(),
	}
}

// SchemaWithSqlOperator sets the sql operator for the schema.
func SchemaWithSqlOperator(self *Schema, sql *SqlOperator) {
	self.sql = sql
}

// SchemaWithNotifier sets the schema notifier.
func SchemaWithNotifier(self *Schema, notifier *Notifier) {
	self.notifier = notifier
}

// SchemaCreateTable creates a table from a type.
func SchemaCreateTable[Table any](self *Schema) {
	if nil == self.sql {
		NotifierSendError(self.notifier, errors.New("schema cannot be created without a sql operator, consider invoking SchemaWithSqlOperator()"))
		return
	}
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
	_, err := self.sql.database.Exec(query.String())
	if err != nil {
		NotifierSendError(self.sql.notifier, err)
	}
}
