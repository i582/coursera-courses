package explorer

type Response struct {
	Response interface{} `json:"response"`
}

type ResponseError struct {
	Response interface{} `json:"error"`
}

type ResponseUpdateRecord struct {
	Updated int64 `json:"updated"`
}

type ResponseAddRecord struct {
	Id int64 `json:"id"`
}

type ResponseRecords struct {
	Records []map[string]interface{} `json:"records"`
}

type ResponseAllTables struct {
	Tables []string `json:"tables"`
}

type ResponseRecord struct {
	Record map[string]interface{} `json:"record"`
}

type ResponseDeleteTable struct {
	Deleted int64 `json:"deleted"`
}
