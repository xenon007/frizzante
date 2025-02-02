package frizzante

import (
	"database/sql"
	"fmt"
	"reflect"
	"strings"
)

func SqlCreateTable(database *sql.DB, instance any) error {
	var query strings.Builder
	t := reflect.TypeOf(instance)
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
	_, err := database.Exec(query.String())
	if err != nil {
		return err
	}
	return nil
}
