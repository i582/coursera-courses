package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/mailru/easyjson"
	"github.com/mailru/easyjson/jlexer"
	"github.com/mailru/easyjson/jwriter"
)

// suppress unused package warning
var (
	_ *json.RawMessage
	_ *jlexer.Lexer
	_ *jwriter.Writer
	_ easyjson.Marshaler
)

func easyjson8be77ed4DecodeHw3BenchFast(in *jlexer.Lexer, out *Record) {
	isTopLevel := in.IsStart()
	if in.IsNull() {
		if isTopLevel {
			in.Consumed()
		}
		in.Skip()
		return
	}
	in.Delim('{')
	for !in.IsDelim('}') {
		key := in.UnsafeFieldName(false)
		in.WantColon()
		if in.IsNull() {
			in.Skip()
			in.WantComma()
			continue
		}
		switch key {
		case "browsers":
			if in.IsNull() {
				in.Skip()
				out.Browsers = nil
			} else {
				in.Delim('[')
				if out.Browsers == nil {
					if !in.IsDelim(']') {
						out.Browsers = make([]string, 0, 4)
					} else {
						out.Browsers = []string{}
					}
				} else {
					out.Browsers = (out.Browsers)[:0]
				}
				for !in.IsDelim(']') {
					var v1 string
					v1 = in.String()
					out.Browsers = append(out.Browsers, v1)
					in.WantComma()
				}
				in.Delim(']')
			}
		case "email":
			out.Email = in.String()
		case "name":
			out.Name = in.String()
		default:
			in.SkipRecursive()
		}
		in.WantComma()
	}
	in.Delim('}')
	if isTopLevel {
		in.Consumed()
	}
}

// UnmarshalJSON supports json.Unmarshaler interface
func (v *Record) UnmarshalJSON(data []byte) error {
	r := jlexer.Lexer{Data: data}
	easyjson8be77ed4DecodeHw3BenchFast(&r, v)
	return r.Error()
}

//easyjson:json
type Record struct {
	Browsers []string `json:"browsers"`
	Email    string   `json:"email"`
	Name     string   `json:"name"`
}

// вам надо написать более быструю оптимальную этой функции
func FastSearch(out io.Writer) {
	file, err := os.Open(filePath)
	if err != nil {
		panic(err)
	}
	reader := bufio.NewReader(file)

	seenBrowsers := make(map[string]struct{}, 100)
	line := make([]byte, 0, 1000)
	users := make([]Record, 0, 1000)

	for {
		line, err = reader.ReadSlice('\n')
		if err == io.EOF {
			break
		}
		var user Record
		err = user.UnmarshalJSON(line)
		if err != nil {
			panic(err)
		}
		users = append(users, user)
	}

	fmt.Fprintln(out, "found users:")

	for userIndex := range users {
		isAndroid := false
		isMSIE := false

		for i := range users[userIndex].Browsers {
			switch {
			case strings.Contains(users[userIndex].Browsers[i], "Android"):
				isAndroid = true
			case strings.Contains(users[userIndex].Browsers[i], "MSIE"):
				isMSIE = true
			default:
				continue
			}

			if _, seenBefore := seenBrowsers[users[userIndex].Browsers[i]]; !seenBefore {
				seenBrowsers[users[userIndex].Browsers[i]] = struct{}{}
			}
		}

		if !isAndroid || !isMSIE {
			continue
		}

		posAt := strings.IndexByte(users[userIndex].Email, '@')
		fmt.Fprintf(out, "[%d] %s <%s [at] %s>\n", userIndex, users[userIndex].Name, users[userIndex].Email[0:posAt], users[userIndex].Email[posAt+1:])
	}

	fmt.Fprintln(out, "\nTotal unique browsers", len(seenBrowsers))
}
