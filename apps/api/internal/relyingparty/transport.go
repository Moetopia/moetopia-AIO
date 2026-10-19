package relyingparty

import (
	"bytes"
	"errors"
	"io"
	"net/http"
)

type protocolTransport struct {
	issuer  string
	handler http.Handler
}

type protocolResponse struct {
	header http.Header
	body   bytes.Buffer
	status int
}

func (w *protocolResponse) Header() http.Header { return w.header }
func (w *protocolResponse) WriteHeader(status int) {
	if w.status == 0 {
		w.status = status
	}
}
func (w *protocolResponse) Write(p []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	if w.body.Len()+len(p) > 1<<20 {
		return 0, errors.New("protocol response too large")
	}
	return w.body.Write(p)
}
func (t protocolTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	if r.URL.Scheme+"://"+r.URL.Host != t.issuer || r.URL.User != nil || r.URL.RawQuery != "" || r.URL.Fragment != "" {
		return nil, errors.New("untrusted RP backchannel destination")
	}
	if !(r.Method == http.MethodPost && r.URL.Path == "/oauth/token" || r.Method == http.MethodGet && r.URL.Path == "/oauth/jwks") {
		return nil, errors.New("unsupported RP backchannel operation")
	}
	if r.Body != nil {
		defer r.Body.Close()
	}
	w := &protocolResponse{header: make(http.Header)}
	t.handler.ServeHTTP(w, r)
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return &http.Response{StatusCode: w.status, Header: w.header, Body: io.NopCloser(bytes.NewReader(w.body.Bytes())), ContentLength: int64(w.body.Len()), Request: r}, nil
}
