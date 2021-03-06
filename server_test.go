package rapid

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

type indexRequest struct {
	ID int
}

type indexResponse struct {
	ID int
}

type testServer struct {
	called bool
	id     int
	params Params
}

func (t *testServer) Index(params Params, req *indexRequest) (*indexResponse, error) {
	t.id = req.ID
	t.called = true
	t.params = params
	return &indexResponse{req.ID * 2}, ErrorForStatus(http.StatusOK)
}

func (t *testServer) Fails() error {
	return ErrorWithHeaders(http.StatusBadRequest, "bad request", http.Header{"X-Error": {"bad request"}})
}

func TestServerMethodDoesNotExist(t *testing.T) {
	svc := Define("Test")
	svc.Route("Invalid", "/").Get().Response(http.StatusOK, nil)
	_, err := NewServer(svc.Build(), &testServer{})
	assert.Error(t, err)
}

func TestServerMethodExists(t *testing.T) {
	svc := Define("Test")
	svc.Route("Index", "/").Get().Response(http.StatusOK, nil)
	_, err := NewServer(svc.Build(), &testServer{})
	assert.NoError(t, err)
}

func TestServerCallsMethod(t *testing.T) {
	svc := Define("Test")
	svc.Route("Index", "/{id}").Get().Request(&indexRequest{}).Response(200, &indexResponse{})

	test := &testServer{}
	svr, _ := NewServer(svc.Build(), test)

	rb := bytes.NewBuffer([]byte(`{"ID": 10}`))
	r, err := http.NewRequest("GET", "/hello", rb)
	assert.NoError(t, err)
	w := httptest.NewRecorder()
	svr.ServeHTTP(w, r)
	assert.True(t, test.called)
	assert.Equal(t, Params{"id": "hello"}, test.params)
	assert.Equal(t, 10, test.id)
	assert.Equal(t, 200, w.Code)
	assert.Equal(t, "{\"ID\":20}\n", w.Body.String())
}

func TestPatternRegex(t *testing.T) {
	svc := Define("Test")
	svc.Route("Index", `/{id:\d\{1,3\}}`).Get().Request(&indexRequest{}).Response(200, &indexResponse{})

	test := &testServer{}
	svr, _ := NewServer(svc.Build(), test)

	rb := bytes.NewBuffer([]byte(`{"ID": 10}`))
	r, _ := http.NewRequest("GET", "/123456", rb)
	w := httptest.NewRecorder()
	svr.ServeHTTP(w, r)
	assert.False(t, test.called)

	rb = bytes.NewBuffer([]byte(`{"ID": 10}`))
	r, _ = http.NewRequest("GET", "/123", rb)
	w = httptest.NewRecorder()
	svr.ServeHTTP(w, r)
	assert.True(t, test.called)
	assert.Equal(t, Params{"id": "123"}, test.params)
}

func TestErrorResponse(t *testing.T) {
	svc := Define("Test")
	svc.Route("Fails", `/`).Get().Response(200, nil)
	test := &testServer{}
	svr, _ := NewServer(svc.Build(), test)
	r, _ := http.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	svr.ServeHTTP(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "bad request", w.Header().Get("X-Error"))
	assert.Equal(t, "{\"e\":\"bad request\"}\n", string(w.Body.Bytes()))
}

type testChunkedServer struct {
	id int
}

func (t *testChunkedServer) Index(params map[string]interface{}) {

}

func TestServerChunkedResponses(t *testing.T) {
	svc := Define("Test")
	svc.Route("Index", "/{id}").Get().Response(200, &indexResponse{})
}

type pathData struct {
	ID int `schema:"id"`
}

type testPathDecodingServer struct {
	id     int
	called bool
}

func (t *testPathDecodingServer) Index(path *pathData) {
	t.id = path.ID
	t.called = true
}

func TestPathDecode(t *testing.T) {
	svc := Define("TestPathDecode")
	svc.Route("Index", "/{id}").Get().Path(&pathData{}).Response(http.StatusOK, nil)

	test := &testPathDecodingServer{}
	svr, _ := NewServer(svc.Build(), test)
	r, _ := http.NewRequest("GET", "/1234", nil)
	w := httptest.NewRecorder()
	svr.ServeHTTP(w, r)
	assert.True(t, test.called)
	assert.Equal(t, 1234, test.id)

	r, _ = http.NewRequest("GET", "/asdf", nil)
	w = httptest.NewRecorder()
	svr.ServeHTTP(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

type queryData struct {
	ID int `schema:"id"`
}

type testQueryDecodingServer struct {
	id     int
	called bool
}

func (t *testQueryDecodingServer) Index(query *queryData) {
	t.id = query.ID
	t.called = true
}

func TestQueryDecode(t *testing.T) {
	svc := Define("TestPathDecode")
	svc.Route("Index", "/").Get().Query(&queryData{}).Response(http.StatusOK, nil)

	test := &testQueryDecodingServer{}
	svr, _ := NewServer(svc.Build(), test)
	r, _ := http.NewRequest("GET", "/?id=1234", nil)
	w := httptest.NewRecorder()
	svr.ServeHTTP(w, r)
	assert.True(t, test.called)
	assert.Equal(t, 1234, test.id)

	r, _ = http.NewRequest("GET", "/?id=asdf", nil)
	w = httptest.NewRecorder()
	svr.ServeHTTP(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestMakeValueAndInterface(t *testing.T) {
	hello := RawData("hello")
	b := bytes.NewReader(hello)
	r, _ := http.NewRequest("POST", "/", b)
	v, i := makeValueAndInterface(reflect.TypeOf(RawData{}))
	c, ok := i.(RequestCodec)
	assert.True(t, ok)
	err := c.DecodeRequest(r)
	assert.NoError(t, err)
	assert.Equal(t, hello, v())
	assert.Equal(t, hello, reflect.ValueOf(i).Elem().Interface())
}

func TestValueAndInterface(t *testing.T) {
	b := RawData("hello")
	_, i := valueAndInterface(b)
	var codec CodecFactory = DefaultCodecFactory
	w := httptest.NewRecorder()
	err := codec.Response(i).EncodeResponse(nil, w, 0, nil)
	assert.NoError(t, err)
	assert.Equal(t, b, RawData(w.Body.Bytes()))
}
