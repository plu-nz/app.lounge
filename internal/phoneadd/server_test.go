package phoneadd

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/plu-nz/app.lounge/internal/stremio"
)

func testOpts(installed *[]string) Options {
	return Options{
		Normalize: stremio.NormalizeURL,
		Install: func(ctx context.Context, u string) (string, error) {
			if strings.Contains(u, "broken") {
				return "", errors.New("manifest unreachable")
			}
			*installed = append(*installed, u)
			return "Test Addon", nil
		},
	}
}

func post(t *testing.T, h http.Handler, path, body string, hdr map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	for k, v := range hdr {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestTokenRequired(t *testing.T) {
	var got []string
	s, _ := New(testOpts(&got))
	h := s.Handler()
	for _, p := range []string{"/", "/wrong/", "/" + s.Token(), "/" + s.Token() + "/x"} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, p, nil))
		if rec.Code != http.StatusNotFound {
			t.Errorf("GET %s = %d, want 404", p, rec.Code)
		}
	}
	if rec := post(t, h, "/wrong/add", `{"url":"https://a.b/manifest.json"}`, nil); rec.Code != http.StatusNotFound {
		t.Errorf("POST wrong token = %d", rec.Code)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/"+s.Token()+"/", nil))
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), "<form") {
		t.Fatalf("page = %d", rec.Code)
	}
	if rec.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Error("CORS header must not be set")
	}
	if len(got) != 0 {
		t.Error("nothing should be installed")
	}
}

func TestValidation(t *testing.T) {
	var got []string
	s, _ := New(testOpts(&got))
	h := s.Handler()
	add := "/" + s.Token() + "/add"
	for _, body := range []string{
		`{"url":"ftp://x.y/manifest.json"}`,
		`{"url":"javascript:alert(1)"}`,
		`{"url":""}`,
		`not json`,
	} {
		if rec := post(t, h, add, body, nil); rec.Code < 400 {
			t.Errorf("%s accepted: %d", body, rec.Code)
		}
	}
	if rec := post(t, h, add, `{"url":"https://broken.example/manifest.json"}`, nil); rec.Code != http.StatusUnprocessableEntity ||
		!strings.Contains(rec.Body.String(), "unreachable") {
		t.Errorf("install error = %d %s", rec.Code, rec.Body)
	}
	if rec := post(t, h, add, `{"url":"x"}`, map[string]string{"Content-Type": "text/plain"}); rec.Code != http.StatusUnsupportedMediaType {
		t.Errorf("content type = %d", rec.Code)
	}
	if rec := post(t, h, add, `{"url":"https://a.b/manifest.json"}`, map[string]string{"Origin": "http://evil.example"}); rec.Code != http.StatusForbidden {
		t.Errorf("foreign origin = %d", rec.Code)
	}
	if rec := post(t, h, add, `{"url":"`+strings.Repeat("a", 10000)+`"}`, nil); rec.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("oversize = %d", rec.Code)
	}
	if len(got) != 0 {
		t.Errorf("installed %v", got)
	}
}

func TestOneTimeUse(t *testing.T) {
	var got []string
	var results []Result
	o := testOpts(&got)
	o.OnResult = func(r Result) { results = append(results, r) }
	s, _ := New(o)
	h := s.Handler()
	add := "/" + s.Token() + "/add"

	// A failed attempt does not consume the token.
	post(t, h, add, `{"url":"https://broken.example/manifest.json"}`, nil)
	rec := post(t, h, add, `{"url":"stremio://good.example/foo"}`, nil)
	if rec.Code != 200 {
		t.Fatalf("success = %d %s", rec.Code, rec.Body)
	}
	if len(got) != 1 || got[0] != "https://good.example/foo/manifest.json" {
		t.Fatalf("installed %v", got)
	}
	if len(results) != 2 || results[0].OK || !results[1].OK || results[1].Name != "Test Addon" {
		t.Fatalf("results %+v", results)
	}
	select {
	case <-s.Done():
	case <-time.After(2 * time.Second):
		t.Fatal("server did not stop after success")
	}
	if rec := post(t, h, add, `{"url":"https://good.example/manifest.json"}`, nil); rec.Code != http.StatusNotFound {
		t.Errorf("second use = %d, want 404", rec.Code)
	}
	if len(got) != 1 {
		t.Errorf("installed twice: %v", got)
	}
}

func TestAttemptLimit(t *testing.T) {
	var got []string
	s, _ := New(testOpts(&got))
	h := s.Handler()
	add := "/" + s.Token() + "/add"
	var last int
	for i := 0; i <= maxAttempts; i++ {
		last = post(t, h, add, `{"url":"https://broken.example/manifest.json"}`, nil).Code
	}
	if last != http.StatusTooManyRequests {
		t.Errorf("after limit = %d", last)
	}
}

func TestTimeoutAndStop(t *testing.T) {
	var got []string
	o := testOpts(&got)
	o.IP = net.ParseIP("127.0.0.1")
	o.TTL = 100 * time.Millisecond
	stops := make(chan struct{}, 2)
	o.OnStop = func() { stops <- struct{}{} }
	s, u, err := Start(o)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.Get(u)
	if err != nil || resp.StatusCode != 200 {
		t.Fatalf("get %v %v", err, resp)
	}
	resp.Body.Close()
	// A wrong Host header is rejected even with the right token.
	req, _ := http.NewRequest(http.MethodGet, u, nil)
	req.Host = "evil.example"
	if resp, err := http.DefaultClient.Do(req); err != nil || resp.StatusCode != 404 {
		t.Errorf("bad host: %v %v", err, resp)
	}
	select {
	case <-s.Done():
	case <-time.After(2 * time.Second):
		t.Fatal("did not time out")
	}
	if _, err := http.Get(u); err == nil {
		t.Error("server still listening after timeout")
	}
	s.Stop() // idempotent
	if len(stops) != 1 {
		t.Errorf("OnStop called %d times", len(stops))
	}
}

func TestDefaultRouteIface(t *testing.T) {
	route := `Iface	Destination	Gateway	Flags	RefCnt	Use	Metric	Mask	MTU	Window	IRTT
wlan0	00000000	0101A8C0	0003	0	0	600	00000000	0	0	0
eth0	00000000	0101A8C0	0003	0	0	100	00000000	0	0	0
docker0	000011AC	00000000	0001	0	0	0	0000FFFF	0	0	0
`
	if got := defaultRouteIface(strings.NewReader(route)); got != "eth0" {
		t.Errorf("got %q, want eth0", got)
	}
	if got := defaultRouteIface(strings.NewReader("Iface\n")); got != "" {
		t.Errorf("got %q", got)
	}
}
