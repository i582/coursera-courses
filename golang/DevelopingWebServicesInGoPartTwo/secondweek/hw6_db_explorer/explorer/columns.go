package explorer

import (
	"database/sql"
	"fmt"
)

type Column struct {
	Name     string
	Type     string
	Nullable bool
}

func TableColumns(db *sql.DB, table *Table) (map[string]*Column, error) {
	if db == nil {
		return nil, fmt.Errorf("db is nil")
	}
	if table == nil {
		return nil, fmt.Errorf("table not set")
	}

	var columns = map[string]*Column{}

	rows, err := db.Query(fmt.Sprintf("SHOW FULL COLUMNS FROM %s", table.Name))
	if err != nil {
		return nil, fmt.Errorf("query error: %v", err)
	}

	for rows.Next() {
		var name string
		var typ string
		var nullable string
		var val interface{}

		err := rows.Scan(&name, &typ, &val, &nullable, &val, &val, &val, &val, &val)
		if err != nil {
			return nil, fmt.Errorf("query error: %v", err)
		}

		columns[name] = &Column{
			Name:     name,
			Type:     typ,
			Nullable: nullable == "YES",
		}
	}

	rows.Close()

	return columns, nil
}
