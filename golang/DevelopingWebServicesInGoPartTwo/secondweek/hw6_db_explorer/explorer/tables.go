package explorer

import (
	"database/sql"
	"fmt"
)

type Table struct {
	Name string `json:"name"`
}

func allTables(db *sql.DB) ([]*Table, error) {
	if db == nil {
		return nil, fmt.Errorf("db is nil")
	}

	var tables []*Table

	rows, err := db.Query("SHOW TABLES")
	if err != nil {
		return nil, fmt.Errorf("query error: %v", err)
	}

	for rows.Next() {
		var name string
		err := rows.Scan(&name)
		if err != nil {
			return nil, fmt.Errorf("query error: %v", err)
		}

		tables = append(tables, &Table{Name: name})
	}

	rows.Close()

	return tables, nil
}
