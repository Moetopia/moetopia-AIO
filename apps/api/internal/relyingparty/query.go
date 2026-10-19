package relyingparty

import (
	"net/http"
	"net/url"
)

func parseQuery(r *http.Request) (url.Values, error) {
	query, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		return nil, ErrLogin
	}
	for _, values := range query {
		if len(values) != 1 {
			return nil, ErrLogin
		}
	}
	return query, nil
}
