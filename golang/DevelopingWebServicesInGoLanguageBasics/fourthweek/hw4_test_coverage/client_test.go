package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

type SearchServer struct {
	wg   *sync.WaitGroup
	data Data
}

type Data struct {
	XMLName xml.Name `xml:"root"`
	Rows    []Row    `xml:"row"`
}

type Row struct {
	XMLName   xml.Name `xml:"row"`
	Id        int64    `xml:"id"`
	FirstName string   `xml:"first_name"`
	LastName  string   `xml:"last_name"`
	Age       int64    `xml:"age"`
	About     string   `xml:"about"`
}

const AccessTokenServer = "1234"

func NewSearchServer() *SearchServer {
	file, err := os.Open("dataset.xml")
	if err != nil {
		panic(err)
	}
	data, err := ioutil.ReadAll(file)
	if err != nil {
		panic(err)
	}
	var res Data

	err = xml.Unmarshal(data, &res)
	if err != nil {
		panic(err)
	}

	return &SearchServer{wg: &sync.WaitGroup{}, data: res}
}

func getIntParam(r *http.Request, name string) (int, bool) {
	raw := r.FormValue(name)
	if raw != "" {
		data, err := strconv.ParseInt(raw, 10, 32)
		if err != nil {
			log.Fatal("error format data:", raw)
		}
		return int(data), true
	}

	return 0, false
}

func (s *SearchServer) Handler(w http.ResponseWriter, r *http.Request) {
	sr := SearchRequest{}
	var results []Row

	at := r.Header.Get("AccessToken")
	if at != AccessTokenServer {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	sr.Limit, _ = getIntParam(r, "limit")
	sr.Offset, _ = getIntParam(r, "offset")
	sr.Query = r.FormValue("query")
	sr.OrderField = r.FormValue("order_field")
	sr.OrderBy, _ = getIntParam(r, "order_by")

	if sr.OrderBy != 0 && sr.OrderBy != 1 && sr.OrderBy != -1 {
		w.WriteHeader(http.StatusBadRequest)
		ser := SearchErrorResponse{Error: "ErrorBadOrderBy"}
		res, _ := json.Marshal(&ser)
		_, _ = fmt.Fprint(w, string(res))
		return
	}

	if sr.Query == "internalError" {
		w.WriteHeader(http.StatusBadRequest)
		ser := SearchErrorResponse{Error: "ErrorBadOrderBy"}
		res, _ := json.Marshal(&ser)
		_, _ = fmt.Fprint(w, string(res)+"10")
		return
	}

	if sr.Query == "serverInternalError" {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if sr.Query == "timeoutError" {
		time.Sleep(1 * time.Second)
		return
	}

	if sr.Query != "" {
		for i, row := range s.data.Rows {
			if i > sr.Limit+sr.Offset {
				continue
			}
			if i < sr.Offset {
				continue
			}
			if strings.Contains(row.About, sr.Query) || strings.Contains(row.FirstName+" "+row.LastName, sr.Query) {
				results = append(results, row)
			}
		}
	} else {
		results = s.data.Rows
	}

	var ErrorBadOrderField bool
	if sr.OrderBy != 0 {
		sort.Slice(results, func(i, j int) bool {
			if ErrorBadOrderField {
				return false
			}
			switch sr.OrderField {
			case "id":
				if sr.OrderBy == -1 {
					return results[i].Id > results[j].Id
				} else {
					return results[i].Id < results[j].Id
				}
			case "age":
				if sr.OrderBy == -1 {
					return results[i].Age > results[j].Age
				} else {
					return results[i].Age < results[j].Age
				}
			case "name", "":
				nameI := results[i].FirstName + results[i].LastName
				nameJ := results[j].FirstName + results[j].LastName
				if sr.OrderBy == -1 {
					return nameI > nameJ
				} else {
					return nameI < nameJ
				}
			default:
				ErrorBadOrderField = true
			}

			return false
		})
	}

	if ErrorBadOrderField {
		w.WriteHeader(http.StatusBadRequest)
		ser := SearchErrorResponse{Error: "ErrorBadOrderField"}
		res, _ := json.Marshal(&ser)
		_, _ = fmt.Fprint(w, string(res))
		return
	}

	res, err := json.Marshal(&results)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}

	if sr.Query == "resultUnpackError" {
		_, _ = fmt.Fprint(w, string(res)+"1")
		return
	}

	_, _ = fmt.Fprint(w, string(res))
}

type TestSuite struct {
	Name          string
	SearchClient  SearchClient
	SearchRequest SearchRequest
	ExpectedError string
}

func SimpleSearchClient(url string) SearchClient {
	return SearchClient{
		AccessToken: AccessTokenServer,
		URL:         url,
	}
}

func TestSimple(t *testing.T) {
	ss := NewSearchServer()
	ts := httptest.NewServer(http.HandlerFunc(ss.Handler))
	defer ts.Close()

	suites := []TestSuite{
		{
			Name: "BadAccessToken",
			SearchClient: SearchClient{
				AccessToken: "",
				URL:         ts.URL,
			},
			SearchRequest: SearchRequest{},
			ExpectedError: "bad AccessToken",
		},
		{
			Name:         "BadLimit",
			SearchClient: SimpleSearchClient(ts.URL),
			SearchRequest: SearchRequest{
				Limit: -10,
			},
			ExpectedError: "limit must be > 0",
		},
		{
			Name:         "LimitBigger25",
			SearchClient: SimpleSearchClient(ts.URL),
			SearchRequest: SearchRequest{
				Limit: 100,
			},
			ExpectedError: "",
		},
		{
			Name:         "BadOffset",
			SearchClient: SimpleSearchClient(ts.URL),
			SearchRequest: SearchRequest{
				Offset: -10,
			},
			ExpectedError: "offset must be > 0",
		},
		{
			Name: "EmptyUrl",
			SearchClient: SearchClient{
				AccessToken: AccessTokenServer,
				URL:         "",
			},
			SearchRequest: SearchRequest{},
			ExpectedError: "unknown error Get \"?limit=1&offset=0&order_by=0&order_field=&query=\": unsupported protocol scheme \"\"",
		},
		{
			Name:         "BadRequest",
			SearchClient: SimpleSearchClient(ts.URL),
			SearchRequest: SearchRequest{
				OrderField: "picture",
				OrderBy:    OrderByDesc,
			},
			ExpectedError: "OrderFeld picture invalid",
		},
		{
			Name:         "RequestLimit",
			SearchClient: SimpleSearchClient(ts.URL),
			SearchRequest: SearchRequest{
				Query:  "a",
				Offset: 10,
				Limit:  24,
			},
			ExpectedError: "",
		},
		{
			Name:         "BadOrderBy",
			SearchClient: SimpleSearchClient(ts.URL),
			SearchRequest: SearchRequest{
				OrderBy: 10,
			},
			ExpectedError: "unknown bad request error: ErrorBadOrderBy",
		},
		{
			Name:         "BadRequestUnpackError",
			SearchClient: SimpleSearchClient(ts.URL),
			SearchRequest: SearchRequest{
				Query: "internalError",
			},
			ExpectedError: "cant unpack error json: invalid character '1' after top-level value",
		},
		{
			Name:         "ServerInternalError",
			SearchClient: SimpleSearchClient(ts.URL),
			SearchRequest: SearchRequest{
				Query: "serverInternalError",
			},
			ExpectedError: "SearchServer fatal error",
		},
		{
			Name:         "ResultUnpackError",
			SearchClient: SimpleSearchClient(ts.URL),
			SearchRequest: SearchRequest{
				Query: "resultUnpackError",
			},
			ExpectedError: "cant unpack result json: invalid character '1' after top-level value",
		},
		{
			Name:         "TimeoutError",
			SearchClient: SimpleSearchClient(ts.URL),
			SearchRequest: SearchRequest{
				Query: "timeoutError",
			},
			ExpectedError: "timeout for limit=1&offset=0&order_by=0&order_field=&query=timeoutError",
		},
	}

	for _, suite := range suites {
		t.Run(suite.Name, func(t *testing.T) {
			_, err := suite.SearchClient.FindUsers(suite.SearchRequest)

			if err != nil && err.Error() != suite.ExpectedError {
				t.Fatalf("not expected error: %v", err)
			}
		})
	}
}
