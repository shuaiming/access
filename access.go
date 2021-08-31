package access

import (
	"bytes"
	"log"
	"net/http"
	"text/template"
	"time"
)

// responseWriter warp http.ResponseWriter.
// There is no way to get http status code and written size by
// using default http.ResponseWriter.
type responseWriter struct {
	http.ResponseWriter
	wroteHeader bool
	status      int
	size        int
}

// WriteHeader warp http.ResponseWriter.WriteHeader
func (w *responseWriter) WriteHeader(s int) {
	if !w.wroteHeader {
		w.wroteHeader = true
		w.status = s
	}

	w.ResponseWriter.WriteHeader(s)
}

// Write warp http.ResponseWriter.Write
func (w *responseWriter) Write(b []byte) (int, error) {

	// look at http.ResponseWriter.Write() implementation
	// 虽然 ResponseWriter.Write() 会保底设置 http.StatusOK，
	// 但是只能调用内部的 ResponseWriter.WriteHeader？导致这里无法
	// 拿到保底状态码 http.StatusOK，日志里会出现HTTP返回码为0的情况。
	// 所以主动调用一下重写的 WriteHeader，修正之。
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}

	size, err := w.ResponseWriter.Write(b)
	w.size += size
	return size, err
}

// Size of http server written bytes
func (w *responseWriter) Size() int {
	return w.size
}

// Status return http server status code
func (w *responseWriter) Status() int {
	return w.status
}

// Access write access log with log.Logger
type Access struct {
	format string
}

type accessValues struct {
	RemoteAddr string
	HTTPMethod string
	URLPath    string
	TimeSpend  int64
	WriteSize  int
	Status     int
}

// New Access
func New(optional ...string) *Access {
	format := "{{.RemoteAddr}} {{.HTTPMethod}} {{.URLPath}} {{.TimeSpend}} {{.WriteSize}} {{.Status}}"

	if len(optional) > 0 {
		format = optional[0]
	}

	return &Access{format}
}

// ServeHTTP implement pod.Handler
func (a *Access) ServeHTTP(
	w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	timeStart := time.Now()

	neww := &responseWriter{w, false, 0, 0}
	httpMethod := r.Method
	urlPath := r.URL.String()

	next(neww, r)

	timeEnd := time.Now()
	timeSpend := timeEnd.Sub(timeStart)

	values := accessValues{
		RemoteAddr: r.RemoteAddr,
		HTTPMethod: httpMethod,
		URLPath:    urlPath,
		TimeSpend:  timeSpend.Milliseconds(),
		WriteSize:  neww.Size(),
		Status:     neww.Status(),
	}

	tmpl, err := template.New("access").Parse(a.format)
	if err != nil {
		log.Println(err)
	}

	var tpl bytes.Buffer
	err = tmpl.Execute(&tpl, values)
	if err != nil {
		log.Println(err)
	}

	log.Println(tpl.String())
}
