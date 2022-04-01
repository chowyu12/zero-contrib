package mux

import (
	"bytes"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/zeromicro/go-zero/rest/httpx"
	"github.com/zeromicro/go-zero/rest/pathvar"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

type mockedResponseWriter struct {
	code int
}

func (m *mockedResponseWriter) Header() http.Header {
	return http.Header{}
}

func (m *mockedResponseWriter) Write(p []byte) (int, error) {
	return len(p), nil
}

func (m *mockedResponseWriter) WriteHeader(code int) {
	m.code = code
}

func TestMuxRouter(t *testing.T) {
	tests := []struct {
		method string
		path   string
		expect bool
		code   int
		err    error
	}{
		// we don't explicitly set status code, framework will do it.
		{http.MethodGet, "/test/{john}/{smith}", true, 200, nil},
		{http.MethodGet, "/a/b/c?a=b", true, 200, nil},
		{http.MethodGet, "/b/d", false, http.StatusNotFound, nil},
	}

	for _, test := range tests {
		t.Run(test.method+":"+test.path, func(t *testing.T) {
			routed := false
			router := NewRouter()

			err := router.Handle(test.method, "/test/{name}/{last_name}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				routed = true
				assert.Equal(t, 2, len(pathvar.Vars(r)))
				w.WriteHeader(200)
			}))
			assert.Nil(t, err)
			err = router.Handle(test.method, "/a/b/c", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				routed = true
				assert.Nil(t, pathvar.Vars(r))
				w.WriteHeader(200)
			}))
			assert.Nil(t, err)
			err = router.Handle(test.method, "/b/c", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				routed = true
				w.WriteHeader(200)
			}))
			assert.Nil(t, err)

			w := new(mockedResponseWriter)
			r, _ := http.NewRequest(test.method, test.path, nil)
			router.ServeHTTP(w, r)

			assert.Equal(t, test.expect, routed)
			assert.Equal(t, test.code, w.code)
		})
	}
}


func TestParseJsonPost(t *testing.T) {
	r, err := http.NewRequest(http.MethodPost, "http://hello.com/mikael/2022?nickname=whatever&zipcode=200000",
		bytes.NewBufferString(`{"location": "shenzhen", "time": 20220225}`))
	assert.Nil(t, err)
	r.Header.Set(httpx.ContentType, httpx.ApplicationJson)

	router := NewRouter()
	err = router.Handle(http.MethodPost, "/{name}/{year}", http.HandlerFunc(func(
		w http.ResponseWriter, r *http.Request) {
		v := struct {
			Name     string `path:"name"`
			Year     int    `path:"year"`
			Nickname string `form:"nickname"`
			Zipcode  int64  `form:"zipcode"`
			Location string `json:"location"`
			Time     int64  `json:"time"`
		}{}

		err = httpx.Parse(r, &v)
		assert.Nil(t, err)
		_, err = io.WriteString(w, fmt.Sprintf("%s:%d:%s:%d:%s:%d", v.Name, v.Year,
			v.Nickname, v.Zipcode, v.Location, v.Time))
		assert.Nil(t, err)
	}))
	assert.Nil(t, err)

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, r)

	assert.Equal(t, "mikael:2022:whatever:200000:shenzhen:20220225", rr.Body.String())
}

func TestParseJsonPostWithIntSlice(t *testing.T) {
	r, err := http.NewRequest(http.MethodPost, "http://hello.com/mikael/2022",
		bytes.NewBufferString(`{"ages": [1, 2], "years": [3, 4]}`))
	assert.Nil(t, err)
	r.Header.Set(httpx.ContentType, httpx.ApplicationJson)

	router := NewRouter()
	err = router.Handle(http.MethodPost, "/{name}/{year}", http.HandlerFunc(func(
		w http.ResponseWriter, r *http.Request) {
		v := struct {
			Name  string  `path:"name"`
			Year  int     `path:"year"`
			Ages  []int   `json:"ages"`
			Years []int64 `json:"years"`
		}{}

		err = httpx.Parse(r, &v)


		assert.Nil(t, err)
		assert.ElementsMatch(t, []int{1, 2}, v.Ages)
		assert.ElementsMatch(t, []int64{3, 4}, v.Years)
	}))
	assert.Nil(t, err)

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, r)
}

func TestParseJsonPostError(t *testing.T) {
	payload := `[{"abcd": "cdef"}]`
	r, err := http.NewRequest(http.MethodPost, "http://hello.com/mikael/2022?nickname=whatever&zipcode=200000",
		bytes.NewBufferString(payload))
	assert.Nil(t, err)
	r.Header.Set(httpx.ContentType, httpx.ApplicationJson)

	router := NewRouter()
	err = router.Handle(http.MethodPost, "/{name}/{year}", http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			v := struct {
				Name     string `path:"name"`
				Year     int    `path:"year"`
				Nickname string `form:"nickname"`
				Zipcode  int64  `form:"zipcode"`
				Location string `json:"location"`
				Time     int64  `json:"time"`
			}{}

			err = httpx.Parse(r, &v)
			assert.NotNil(t, err)
		}))
	assert.Nil(t, err)

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, r)
}

func TestParsePath(t *testing.T) {
	r, err := http.NewRequest(http.MethodGet, "http://hello.com/mikael/2022", nil)
	assert.Nil(t, err)

	router := NewRouter()
	err = router.Handle(http.MethodGet, "/{name}/{year}", http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			v := struct {
				Name string `path:"name"`
				Year int    `path:"year"`
			}{}

			err = httpx.Parse(r, &v)
			assert.Nil(t, err)
			_, err = io.WriteString(w, fmt.Sprintf("%s in %d", v.Name, v.Year))
			assert.Nil(t, err)
		}))
	assert.Nil(t, err)

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, r)

	assert.Equal(t, "mikael in 2022", rr.Body.String())
}

func TestParsePathRequired(t *testing.T) {
	r, err := http.NewRequest(http.MethodGet, "http://hello.com/mikael", nil)
	assert.Nil(t, err)

	router := NewRouter()
	err = router.Handle(http.MethodGet, "/{name}/", http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			v := struct {
				Name string `path:"name"`
				Year int    `path:"year"`
			}{}

			err = httpx.Parse(r, &v)
			assert.NotNil(t, err)
		}))
	assert.Nil(t, err)

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, r)
}

func TestParseQuery(t *testing.T) {
	r, err := http.NewRequest(http.MethodGet, "http://hello.com/mikael/2022?nickname=whatever&zipcode=200000", nil)
	assert.Nil(t, err)

	router := NewRouter()
	err = router.Handle(http.MethodGet, "/{name}/{year}", http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			v := struct {
				Nickname string `form:"nickname"`
				Zipcode  int64  `form:"zipcode"`
			}{}

			err = httpx.Parse(r, &v)
			assert.Nil(t, err)
			_, err = io.WriteString(w, fmt.Sprintf("%s:%d", v.Nickname, v.Zipcode))
			assert.Nil(t, err)
		}))
	assert.Nil(t, err)

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, r)

	assert.Equal(t, "whatever:200000", rr.Body.String())
}

func TestParseOptional(t *testing.T) {
	r, err := http.NewRequest(http.MethodGet, "http://hello.com/mikael/2022?nickname=whatever&zipcode=", nil)
	assert.Nil(t, err)

	router := NewRouter()
	err = router.Handle(http.MethodGet, "/{name}/{year}", http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			v := struct {
				Nickname string `form:"nickname"`
				Zipcode  int64  `form:"zipcode,optional"`
			}{}

			err = httpx.Parse(r, &v)
			assert.Nil(t, err)
			_, err = io.WriteString(w, fmt.Sprintf("%s:%d", v.Nickname, v.Zipcode))
			assert.Nil(t, err)
		}))
	assert.Nil(t, err)

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, r)

	assert.Equal(t, "whatever:0", rr.Body.String())
}


func BenchmarkMuxRouter(b *testing.B) {
	b.ReportAllocs()

	router := NewRouter()
	router.Handle(http.MethodGet, "/api/param/{param1}/{params2}/{param3}/{param4}/{param5}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	}))
	w := &mockedResponseWriter{}
	r, _ := http.NewRequest(http.MethodGet, "/api/param/path/to/parameter/john/12345", nil)
	for i := 0; i < b.N; i++ {
		router.ServeHTTP(w, r)
	}
}
