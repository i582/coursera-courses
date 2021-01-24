package explorer

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"

	"hw6_db_explorer/router"
)

// тут вы пишете код
// обращаю ваше внимание - в этом задании запрещены глобальные переменные
type DbExplorer struct {
	db *sql.DB
}

func NewDbExplorer(db *sql.DB) (*DbExplorer, error) {
	return &DbExplorer{
		db: db,
	}, nil
}

func (d *DbExplorer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rout := router.New()
	rout.HandleFunc("/", d.tablesHandler)
	rout.HandleFunc("/:table", d.tableHandler)
	rout.HandleFunc("/:table/:id", d.tableRecordHandler)

	rout.ServeHTTP(w, r)
}

func (d *DbExplorer) tablesHandler(w http.ResponseWriter, r *http.Request) {
	d.showAllTables(w, r)
}

func (d *DbExplorer) tableHandler(w http.ResponseWriter, r *http.Request) {
	paramsMap := r.Context().Value("params")
	if paramsMap == nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	params := paramsMap.(map[string]string)
	if _, has := params["table"]; !has {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "table not set")
		return
	}

	table := &Table{Name: params["table"]}

	switch r.Method {
	case http.MethodGet:
		d.showRecords(table, w, r)
	case http.MethodPut:
		d.addRecord(table, w, r)
	}
}

func (d *DbExplorer) tableRecordHandler(w http.ResponseWriter, r *http.Request) {
	paramsMap := r.Context().Value("params")
	if paramsMap == nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	var params = paramsMap.(map[string]string)

	tableName, has := params["table"]
	if !has {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "table not set")
		return
	}

	strId, has := params["id"]
	if !has {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "id not set")
		return
	}

	recordId, err := strconv.ParseInt(strId, 0, 64)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "id not number")
		return
	}

	table := &Table{Name: tableName}

	switch r.Method {
	case http.MethodGet:
		d.showRecord(table, recordId, w)
	case http.MethodDelete:
		d.deleteRecord(table, recordId, w)
	case http.MethodPost:
		d.updateRecord(table, recordId, w, r)
	}
}

func (d *DbExplorer) addRecord(table *Table, w http.ResponseWriter, r *http.Request) {
	params := d.parseForm(r, w)

	id, primary, err := addRecordToTable(d.db, table, params)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "internal server error: %v", err)
		return
	}

	var data = map[string]interface{}{}
	data[primary] = id

	jsonData, err := json.Marshal(Response{data})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "internal server error: %v", err)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, string(jsonData))
}

func (d *DbExplorer) showRecord(table *Table, id int64, w http.ResponseWriter) {
	record, err := recordById(d.db, table, id)
	if err != nil {
		if err.Error() == RecordNotFound {
			w.WriteHeader(http.StatusNotFound)
			jsonData, err := json.Marshal(ResponseError{Response: RecordNotFound})
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprintf(w, "internal server error: %v", err)
				return
			}

			fmt.Fprint(w, string(jsonData))
			return
		}

		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "%v", err)
		return
	}

	jsonData, err := json.Marshal(Response{ResponseRecord{record.Vals}})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "internal server error: %v", err)
		return
	}

	fmt.Fprint(w, string(jsonData))
}

func (d *DbExplorer) updateRecord(table *Table, id int64, w http.ResponseWriter, r *http.Request) {
	data := d.parseForm(r, w)

	aff, err := updateRecordById(d.db, table, data, id)
	if err != nil {
		if strings.Contains(err.Error(), InvalidType) {
			w.WriteHeader(http.StatusBadRequest)
			jsonData, err := json.Marshal(ResponseError{Response: err.Error()})
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprintf(w, "internal server error: %v", err)
				return
			}

			fmt.Fprint(w, string(jsonData))
			return
		}

		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "internal server error: %v", err)
		return
	}

	jsonData, err := json.Marshal(Response{ResponseUpdateRecord{aff}})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "internal server error: %v", err)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, string(jsonData))
}

func (d *DbExplorer) deleteRecord(table *Table, id int64, w http.ResponseWriter) {
	deleted, err := deleteRecordById(d.db, table, id)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "bad requst: %v", err)
		return
	}

	jsonData, err := json.Marshal(Response{ResponseDeleteTable{deleted}})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "internal server error: %v", err)
		return
	}

	fmt.Fprint(w, string(jsonData))
}

func (d *DbExplorer) showRecords(table *Table, w http.ResponseWriter, r *http.Request) {
	limit, err := strconv.ParseInt(r.FormValue("limit"), 0, 64)
	if err != nil {
		limit = math.MaxInt64
	}
	offset, err := strconv.ParseInt(r.FormValue("offset"), 0, 64)
	if err != nil {
		offset = 0
	}

	records, err := recordsFromTable(d.db, table, offset, limit)
	if err != nil {
		if err.Error() == TableNotFound {
			w.WriteHeader(http.StatusNotFound)
			jsonData, err := json.Marshal(ResponseError{Response: "unknown table"})
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprintf(w, "internal server error: %v", err)
				return
			}

			fmt.Fprint(w, string(jsonData))
			return
		}

		fmt.Fprintf(w, "%v", err)
		return
	}

	var recordsView []map[string]interface{}
	for _, record := range records {
		recordsView = append(recordsView, record.Vals)
	}

	jsonData, err := json.Marshal(Response{ResponseRecords{recordsView}})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "internal server error: %v", err)
		return
	}

	fmt.Fprint(w, string(jsonData))
}

func (d *DbExplorer) showAllTables(w http.ResponseWriter, r *http.Request) {
	tables, err := allTables(d.db)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "internal server error: %v", err)
		return
	}

	responseTables := ResponseAllTables{}
	for _, table := range tables {
		responseTables.Tables = append(responseTables.Tables, table.Name)
	}

	jsonData, err := json.Marshal(Response{responseTables})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "internal server error: %v", err)
		return
	}

	fmt.Fprint(w, string(jsonData))
}

func (d *DbExplorer) parseForm(r *http.Request, w http.ResponseWriter) map[string]interface{} {
	err := r.ParseForm()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "internal server error: %v", err)
		return nil
	}

	params := make(map[string]interface{})

	if r.Header.Get("Content-Type") == "application/json" {
		var v interface{}
		err := json.NewDecoder(r.Body).Decode(&v)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "internal server error: %v", err)
			return nil
		}

		return v.(map[string]interface{})
	}

	for key, param := range r.Form {
		params[key] = param[0]
	}

	return params
}
