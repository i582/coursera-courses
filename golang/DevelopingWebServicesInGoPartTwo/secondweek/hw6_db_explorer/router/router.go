package router

import (
	"context"
	"net/http"
	"strings"
)

type path struct {
	handler func(http.ResponseWriter, *http.Request)
	path    []patternPart
}

type Router struct {
	paths map[string]path
}

func New() *Router {
	return &Router{
		paths: map[string]path{},
	}
}

func (r *Router) HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request)) {
	parts := parsePattern(pattern)

	r.paths[pattern] = path{
		handler: handler,
		path:    parts,
	}
}

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	uri := req.RequestURI
	uriPaths := parseUri(uri)

	var handler func(http.ResponseWriter, *http.Request)
	var handlerPath path
	var maxFirstDynamic int

	for _, path := range r.paths {
		compatible, firstDynamic := uriCompatibleWithPattern(uriPaths, path.path)
		if compatible && firstDynamic >= maxFirstDynamic {
			maxFirstDynamic = firstDynamic
			handler = path.handler
			handlerPath = path
		}
	}
	if handler == nil {
		return
	}

	values := dynamicValueFromUri(uriPaths, handlerPath.path)
	req = req.WithContext(context.WithValue(req.Context(), "params", values))
	handler(w, req)
}

func dynamicValueFromUri(uri, pattern []patternPart) map[string]string {
	res := make(map[string]string, len(uri))

	for i, part := range uri {
		if !pattern[i].dynamic {
			continue
		}

		res[pattern[i].value[1:]] = part.value
	}

	return res
}

func uriCompatibleWithPattern(uri, pattern []patternPart) (bool, int) {
	if len(uri) != len(pattern) {
		return false, 0
	}

	var firstDynamic int

	for i, part := range uri {
		if firstDynamic == 0 && pattern[i].dynamic {
			firstDynamic = i + 1
		}

		if !pattern[i].dynamic && pattern[i].value != part.value {
			return false, firstDynamic
		}
	}

	return true, firstDynamic
}

type patternPart struct {
	value   string
	dynamic bool
}

func parsePattern(pattern string) []patternPart {
	pattern = strings.TrimPrefix(pattern, "/")
	pattern = strings.TrimSuffix(pattern, "/")

	rawParts := strings.Split(pattern, "/")
	parts := make([]patternPart, 0, len(rawParts))

	for _, part := range rawParts {
		if len(part) == 0 {
			continue
		}

		parts = append(parts, patternPart{
			value:   part,
			dynamic: part[0] == ':',
		})
	}

	return parts
}

func parseUri(pattern string) []patternPart {
	pattern = strings.TrimPrefix(pattern, "/")
	pattern = strings.TrimSuffix(pattern, "/")
	pattern = strings.TrimSuffix(pattern, "?")

	rawParts := strings.Split(pattern, "/")
	parts := make([]patternPart, 0, len(rawParts))

	for _, part := range rawParts {
		if len(part) == 0 {
			continue
		}

		if index := strings.IndexRune(part, '?'); index != -1 {
			part = part[:index]
		}

		parts = append(parts, patternPart{
			value:   part,
			dynamic: part[0] == ':',
		})
	}

	return parts
}
