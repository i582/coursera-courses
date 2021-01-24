package explorer

import (
	"database/sql"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
)

const RecordNotFound = "record not found"
const TableNotFound = "table not found"
const EmptyData = "empty data"
const InvalidType = "have invalid type"

type Record struct {
	Fields []string
	Vals   map[string]interface{}
}

func typesIsCompatible(col *Column, field interface{}) bool {
	needIsInteger := strings.Contains(col.Type, "int")
	needIsString := strings.Contains(col.Type, "text") || strings.Contains(col.Type, "varchar")
	needIsNullable := col.Nullable

	switch field.(type) {
	case float64:
		if !needIsInteger {
			return false
		}
	case string:
		if !needIsString {
			return false
		}
	case nil:
		if !needIsNullable {
			return false
		}
	}

	return true
}

func recordById(db *sql.DB, table *Table, id int64) (*Record, error) {
	if db == nil {
		return nil, fmt.Errorf("db is nil")
	}
	if table == nil {
		return nil, fmt.Errorf("table not set")
	}

	primary, err := getPrimaryKeyForTable(db, table)
	if err != nil {
		return nil, fmt.Errorf("query error: %v", err)
	}

	rows, err := db.Query(fmt.Sprintf("SELECT * FROM %s WHERE %s=%d", table.Name, primary, id))
	if err != nil {
		return nil, fmt.Errorf("query error: %v", err)
	}

	var countRows int
	var record *Record

	for rows.Next() {
		countRows++
		record, err = handleRecordRawData(rows)
		if err != nil {
			return nil, err
		}
	}

	rows.Close()

	if countRows == 0 {
		return nil, errors.New(RecordNotFound)
	}

	return record, nil
}

func addRecordToTable(db *sql.DB, table *Table, data map[string]interface{}) (int64, string, error) {
	if db == nil {
		return -1, "", errors.New("db is nil")
	}
	if table == nil {
		return -1, "", errors.New("table not set")
	}
	if len(data) == 0 {
		return -1, "", errors.New(EmptyData)
	}

	primary, err := getPrimaryKeyForTable(db, table)
	if err != nil {
		return -1, "", fmt.Errorf("query error: %v", err)
	}

	cols, err := TableColumns(db, table)
	if err != nil {
		return -1, "", fmt.Errorf("query error: %v", err)
	}

	var fields string
	var values string

	for field, col := range cols {
		if field == primary {
			continue
		}

		fields += field + ", "

		value, has := data[field]
		if !has {
			if col.Nullable {
				values += `NULL, `
				continue
			}

			values += `"", `
			continue
		}

		switch value.(type) {
		case float64:
			values += fmt.Sprint(value)
		case string:
			values += strconv.Quote(fmt.Sprint(value))
		}

		values += ", "
	}

	fields = strings.TrimSuffix(fields, ", ")
	values = strings.TrimSuffix(values, ", ")

	res, err := db.Exec(fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", table.Name, fields, values))
	if err != nil {
		return -1, "", fmt.Errorf("query error: %v", err)
	}

	_, err = res.RowsAffected()
	if err != nil {
		return -1, "", fmt.Errorf("query error: %v", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return -1, "", fmt.Errorf("query error: %v", err)
	}

	return id, primary, nil
}

func updateRecordById(db *sql.DB, table *Table, data map[string]interface{}, id int64) (int64, error) {
	if db == nil {
		return 0, fmt.Errorf("db is nil")
	}
	if table == nil {
		return 0, fmt.Errorf("table not set")
	}

	primary, err := getPrimaryKeyForTable(db, table)
	if err != nil {
		return 0, fmt.Errorf("query error: %v", err)
	}

	cols, err := TableColumns(db, table)
	if err != nil {
		return 0, fmt.Errorf("query error: %v", err)
	}

	var fieldsWithValues string
	for key, field := range data {
		if key == primary {
			return 0, fmt.Errorf("field %s have invalid type", key)
		}

		fieldsWithValues += key + "="

		if !typesIsCompatible(cols[key], field) {
			return 0, fmt.Errorf("field %s have invalid type", key)
		}

		switch field.(type) {
		case float64:
			fieldsWithValues += fmt.Sprint(field)
		case string:
			fieldsWithValues += "'" + fmt.Sprint(field) + "'"
		case nil:
			fieldsWithValues += "NULL"
		}

		fieldsWithValues += ", "
	}

	fieldsWithValues = strings.TrimSuffix(fieldsWithValues, ", ")

	res, err := db.Exec(fmt.Sprintf("UPDATE %s SET %s WHERE %s=%d", table.Name, fieldsWithValues, primary, id))
	if err != nil {
		return 0, fmt.Errorf("query error: %v", err)
	}

	aff, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("query error: %v", err)
	}

	return aff, nil
}

func deleteRecordById(db *sql.DB, table *Table, id int64) (int64, error) {
	if db == nil {
		return 0, fmt.Errorf("db is nil")
	}
	if table == nil {
		return 0, fmt.Errorf("table not set")
	}

	primary, err := getPrimaryKeyForTable(db, table)
	if err != nil {
		return 0, fmt.Errorf("query error: %v", err)
	}

	res, err := db.Exec(fmt.Sprintf("DELETE FROM %s WHERE %s=%d", table.Name, primary, id))
	if err != nil {
		return 0, fmt.Errorf("query error: %v", err)
	}

	aff, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("query error: %v", err)
	}

	return aff, nil
}

func recordsFromTable(db *sql.DB, table *Table, offset, limit int64) ([]*Record, error) {
	if db == nil {
		return nil, fmt.Errorf("db is nil")
	}
	if table == nil {
		return nil, fmt.Errorf("table not set")
	}
	if limit == 0 {
		limit = math.MaxInt64
	}

	rows, err := db.Query(fmt.Sprintf("SELECT * FROM `%s` LIMIT %d OFFSET %d", table.Name, limit, offset))
	if err != nil {
		if strings.Contains(err.Error(), ` 1146`) {
			return nil, errors.New(TableNotFound)
		}

		return nil, fmt.Errorf("query error: %v", err)
	}

	var records []*Record

	for rows.Next() {
		record, err := handleRecordRawData(rows)
		if err != nil {
			return nil, err
		}

		records = append(records, record)
	}

	rows.Close()

	return records, nil
}

func handleRecordRawData(rows *sql.Rows) (*Record, error) {
	cols, _ := rows.Columns()
	colsTypes, _ := rows.ColumnTypes()

	var vals = make(map[string]interface{})
	var values = make([]interface{}, len(cols))
	for i := range values {
		values[i] = &values[i]
	}

	err := rows.Scan(values...)
	if err != nil {
		return nil, fmt.Errorf("query error: %v", err)
	}

	for i := range values {
		colType := colsTypes[i]
		switch colType.ScanType().Name() {
		case "int32":
			col := cols[i]
			vals[col], err = strconv.ParseInt(string(values[i].([]byte)), 0, 64)
		case "RawBytes":
			col := cols[i]
			if nullable, ok := colType.Nullable(); nullable && ok {
				if values[i] == nil {
					vals[col] = nil
				} else {
					vals[col] = string(values[i].([]byte))
				}
			} else {
				vals[col] = string(values[i].([]byte))
			}
		default:
			fmt.Println(colType.ScanType().Name())
		}
	}
	return &Record{
		Fields: cols,
		Vals:   vals,
	}, nil
}

func getPrimaryKeyForTable(db *sql.DB, table *Table) (string, error) {
	res, err := db.Query(fmt.Sprintf("SHOW KEYS FROM %s WHERE Key_name = 'PRIMARY'", table.Name))
	if err != nil {
		return "", fmt.Errorf("query error: %v", err)
	}

	var primaryKey string

	for res.Next() {
		cols, _ := res.Columns()

		var values = make([]interface{}, len(cols))
		for i := range values {
			values[i] = &values[i]
		}

		err := res.Scan(values...)
		if err != nil {
			return "", fmt.Errorf("query error: %v", err)
		}

		primaryKey = string(values[4].([]byte))
	}

	return primaryKey, nil
}
