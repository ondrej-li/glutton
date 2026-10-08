package handler_test

import (
	"errors"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/defectus/glutton/pkg/common"
	"github.com/defectus/glutton/pkg/handler"
	"github.com/defectus/glutton/pkg/iface"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockSaver struct {
	mock.Mock
}

func (m *MockSaver) Save(payload *iface.PayloadRecord) error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockSaver) Configure(*iface.Settings) error {
	return nil
}

type TestSaver struct {
}

func (t *TestSaver) Save(payload *iface.PayloadRecord) error {
	log.Printf("TestSaver_Save called with %+v", payload)
	return nil
}

func (t *TestSaver) Configure(settings *iface.Settings) error {
	log.Printf("TestSaver_Configure called with %+v", settings)
	return nil
}

type MockParser struct {
	mock.Mock
}

func (m *MockParser) Configure(*iface.Settings) error {
	return nil
}

func (m *MockParser) Parse(request *http.Request) (*iface.PayloadRecord, error) {
	args := m.Called()
	return args.Get(0).(*iface.PayloadRecord), args.Error(1)
}

type TestParser struct {
}

func (t *TestParser) Configure(settings *iface.Settings) error {
	log.Printf("TestParser_Configure called with %+v", settings)
	return nil
}

func (t *TestParser) Parse(request *http.Request) (*iface.PayloadRecord, error) {
	log.Printf("TestParser_Parse called with %+v", request)
	return &iface.PayloadRecord{}, nil
}

type MockNotifier struct {
	mock.Mock
}

func (m *MockNotifier) Configure(*iface.Settings) error {
	return nil
}

func (m *MockNotifier) Notify(*iface.PayloadRecord) error {
	args := m.Called()
	return args.Error(0)
}

type TestNotifier struct {
}

func (t *TestNotifier) Configure(settings *iface.Settings) error {
	log.Printf("TestNotifier_Configure called with %+v", settings)
	return nil
}

func (t *TestNotifier) Notify(payload *iface.PayloadRecord) error {
	log.Printf("TestNotifier_Notify called with %+v", payload)
	return nil
}

func TestCreateHandler(t *testing.T) {
	mp := &MockParser{}
	mp.On("Parse").Return(&iface.PayloadRecord{}, nil)
	ms := &MockSaver{}
	ms.On("Save").Return(nil)
	mn := &MockNotifier{}
	mn.On("Notify").Return(nil)
	router := gin.Default()
	router.POST("test", handler.CreateHandler("test", mp, mn, ms, false))
	req, _ := http.NewRequest("POST", "http://localhost/test", nil)
	testHTTPResponse(t, router, req, func(w *httptest.ResponseRecorder) bool {
		assert.Equal(t, http.StatusOK, w.Code)
		p, _ := io.ReadAll(w.Body)
		log.Printf("server reply: %s", string(p))
		mp.AssertExpectations(t)
		ms.AssertExpectations(t)
		mn.AssertExpectations(t)
		return true
	})
}

func TestCreateHandlerParseError(t *testing.T) {
	mp := &MockParser{}
	mp.On("Parse").Return((*iface.PayloadRecord)(nil), errors.New("parse failed"))
	ms := &MockSaver{}
	mn := &MockNotifier{}
	router := gin.Default()
	router.POST("test", handler.CreateHandler("test", mp, mn, ms, false))
	req, _ := http.NewRequest("POST", "http://localhost/test", nil)
	testHTTPResponse(t, router, req, func(w *httptest.ResponseRecorder) bool {
		assert.Equal(t, http.StatusBadRequest, w.Code)
		mn.AssertNotCalled(t, "Notify")
		ms.AssertNotCalled(t, "Save")
		return true
	})
}

func TestCreateHandlerPayloadTooLarge(t *testing.T) {
	mp := &MockParser{}
	mp.On("Parse").Return((*iface.PayloadRecord)(nil), iface.ErrPayloadTooLarge)
	ms := &MockSaver{}
	mn := &MockNotifier{}
	router := gin.Default()
	router.POST("test", handler.CreateHandler("test", mp, mn, ms, false))
	req, _ := http.NewRequest("POST", "http://localhost/test", nil)
	testHTTPResponse(t, router, req, func(w *httptest.ResponseRecorder) bool {
		return assert.Equal(t, http.StatusRequestEntityTooLarge, w.Code)
	})
}

func TestCreateHandlerNotifierError(t *testing.T) {
	mp := &MockParser{}
	mp.On("Parse").Return(&iface.PayloadRecord{}, nil)
	mn := &MockNotifier{}
	mn.On("Notify").Return(errors.New("notify failed"))
	ms := &MockSaver{}
	ms.On("Save").Return(nil)
	router := gin.Default()
	router.POST("test", handler.CreateHandler("test", mp, mn, ms, false))
	req, _ := http.NewRequest("POST", "http://localhost/test", nil)
	testHTTPResponse(t, router, req, func(w *httptest.ResponseRecorder) bool {
		// the payload is still handed to the saver
		ms.AssertExpectations(t)
		return assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestCreateHandlerSaverError(t *testing.T) {
	mp := &MockParser{}
	mp.On("Parse").Return(&iface.PayloadRecord{}, nil)
	mn := &MockNotifier{}
	mn.On("Notify").Return(nil)
	ms := &MockSaver{}
	ms.On("Save").Return(errors.New("save failed"))
	router := gin.Default()
	router.POST("test", handler.CreateHandler("test", mp, mn, ms, false))
	req, _ := http.NewRequest("POST", "http://localhost/test", nil)
	testHTTPResponse(t, router, req, func(w *httptest.ResponseRecorder) bool {
		return assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestCreateRedirectHandlerNoRedirect(t *testing.T) {
	mp := &MockParser{}
	mp.On("Parse").Return(&iface.PayloadRecord{}, nil)
	ms := &MockSaver{}
	ms.On("Save").Return(nil)
	mn := &MockNotifier{}
	mn.On("Notify").Return(nil)
	router := gin.Default()
	router.POST("test", handler.RedirectHandler(handler.CreateHandler("test", mp, mn, ms, false), http.StatusTemporaryRedirect, ""))
	req, _ := http.NewRequest("POST", "http://localhost/test", nil)
	testHTTPResponse(t, router, req, func(w *httptest.ResponseRecorder) bool {
		assert.Equal(t, http.StatusOK, w.Code)
		p, _ := io.ReadAll(w.Body)
		log.Printf("server reply: %s", string(p))
		mp.AssertExpectations(t)
		ms.AssertExpectations(t)
		mn.AssertExpectations(t)
		return true
	})
}

func TestCreateRedirectHandlerRedirect(t *testing.T) {
	mp := &MockParser{}
	mp.On("Parse").Return(&iface.PayloadRecord{}, nil)
	ms := &MockSaver{}
	ms.On("Save").Return(nil)
	mn := &MockNotifier{}
	mn.On("Notify").Return(nil)
	router := gin.Default()
	router.POST("test", handler.RedirectHandler(handler.CreateHandler("test", mp, mn, ms, false), http.StatusTemporaryRedirect, "https://test.redirect"))
	req, _ := http.NewRequest("POST", "http://localhost/test", nil)
	testHTTPResponse(t, router, req, func(w *httptest.ResponseRecorder) bool {
		assert.Equal(t, http.StatusTemporaryRedirect, w.Code)
		p, _ := io.ReadAll(w.Body)
		log.Printf("server reply: %s", string(p))
		log.Printf("server header: %+v", w.HeaderMap)
		mp.AssertExpectations(t)
		ms.AssertExpectations(t)
		mn.AssertExpectations(t)
		return true
	})
}

func TestRedirectHandlerKeepsRejectionStatus(t *testing.T) {
	mp := &MockParser{}
	ms := &MockSaver{}
	mn := &MockNotifier{}
	router := gin.Default()
	router.POST("test", handler.RedirectHandler(
		handler.ValidateTokenHandler(handler.CreateHandler("test", mp, mn, ms, false), "test", []byte(""), false),
		http.StatusTemporaryRedirect, "https://test.redirect"))
	req, _ := http.NewRequest("POST", "http://localhost/test", nil)
	req.Header.Add("token", "not-a-valid-token")
	testHTTPResponse(t, router, req, func(w *httptest.ResponseRecorder) bool {
		return assert.Equal(t, http.StatusPreconditionFailed, w.Code) &&
			assert.Empty(t, w.Header().Get("Location"))
	})
}

func TestRedirectHandlerSkipsRedirectOnFailureStatus(t *testing.T) {
	router := gin.Default()
	router.POST("test", handler.RedirectHandler(func(c *gin.Context) {
		c.Status(http.StatusInternalServerError)
	}, http.StatusTemporaryRedirect, "https://test.redirect"))
	req, _ := http.NewRequest("POST", "http://localhost/test", nil)
	testHTTPResponse(t, router, req, func(w *httptest.ResponseRecorder) bool {
		return assert.Equal(t, http.StatusInternalServerError, w.Code) &&
			assert.Empty(t, w.Header().Get("Location"))
	})
}

func TestGenerateTokenHandlerFailsOnInvalidKey(t *testing.T) {
	// a key that is not a valid AES length makes token generation fail; the endpoint must not answer with an empty 200
	router := gin.Default()
	router.GET("token", handler.CreateTokenHandler("test", []byte("short"), false))
	req, _ := http.NewRequest("GET", "http://localhost/token", nil)
	req.Header.Add(handler.TokenKeyHeader, "short")
	testHTTPResponse(t, router, req, func(w *httptest.ResponseRecorder) bool {
		body, _ := io.ReadAll(w.Body)
		return assert.Equal(t, http.StatusInternalServerError, w.Code) &&
			assert.Empty(t, body)
	})
}

func TestGenerateToken(t *testing.T) {
	env := common.CreateEnvironment(&iface.Configuration{
		Debug: true,
		Settings: []iface.Settings{
			{
				UseToken:     true,
				URI:          "save",
				TokenKey:     "0123456789abcdef",
				OutputFolder: t.TempDir(),
			},
		},
	}, nil)
	req, _ := http.NewRequest("GET", "http://localhost/v1/glutton/save/token", nil)
	req.Header.Add(handler.TokenKeyHeader, "0123456789abcdef")
	testHTTPResponse(t, env.Server, req, func(w *httptest.ResponseRecorder) bool {
		assert.Equal(t, http.StatusOK, w.Code)
		p, _ := io.ReadAll(w.Body)
		log.Printf("server reply: %s", string(p))
		log.Printf("server header: %+v", w.HeaderMap)
		return true
	})
}

func TestGenerateTokenRequiresKey(t *testing.T) {
	env := common.CreateEnvironment(&iface.Configuration{
		Debug: true,
		Settings: []iface.Settings{
			{
				UseToken:     true,
				URI:          "save",
				TokenKey:     "0123456789abcdef",
				OutputFolder: t.TempDir(),
			},
		},
	}, nil)
	req, _ := http.NewRequest("GET", "http://localhost/v1/glutton/save/token", nil)
	testHTTPResponse(t, env.Server, req, func(w *httptest.ResponseRecorder) bool {
		return assert.Equal(t, http.StatusPreconditionFailed, w.Code)
	})
}

func TestValidateToken1(t *testing.T) {
	// test fail token (token invalid)
	env := &iface.Env{
		Savers:    map[string]reflect.Type{},
		Notifiers: map[string]reflect.Type{},
		Parsers:   map[string]reflect.Type{},
	}
	env.Notifiers["TestNotifier"] = reflect.TypeFor[TestNotifier]()
	env.Savers["TestSaver"] = reflect.TypeFor[TestSaver]()
	env.Parsers["TestParser"] = reflect.TypeFor[TestParser]()
	env = common.CreateEnvironment(&iface.Configuration{
		Debug: true,
		Settings: []iface.Settings{
			{
				UseToken: true,
				URI:      "save",
				TokenKey: "0123456789abcdef",
				Saver:    "TestSaver",
				Parser:   "TestParser",
				Notifier: "TestNotifier",
			},
		},
	}, env)
	req, _ := http.NewRequest("POST", "http://localhost/v1/glutton/save", nil)
	req.Header.Add("token", "dummy")
	testHTTPResponse(t, env.Server, req, func(w *httptest.ResponseRecorder) bool {
		assert.Equal(t, http.StatusPreconditionFailed, w.Code)
		p, _ := io.ReadAll(w.Body)
		log.Printf("server reply: %s", string(p))
		log.Printf("server header: %+v", w.HeaderMap)
		return true
	})
}

func TestValidateToken2(t *testing.T) {
	// test get token and use token
	env := &iface.Env{
		Savers:    map[string]reflect.Type{},
		Notifiers: map[string]reflect.Type{},
		Parsers:   map[string]reflect.Type{},
	}
	env.Notifiers["TestNotifier"] = reflect.TypeFor[TestNotifier]()
	env.Savers["TestSaver"] = reflect.TypeFor[TestSaver]()
	env.Parsers["TestParser"] = reflect.TypeFor[TestParser]()
	env = common.CreateEnvironment(&iface.Configuration{
		Debug: true,
		Settings: []iface.Settings{
			{
				UseToken: true,
				URI:      "save",
				TokenKey: "0123456789abcdef",
				Saver:    "TestSaver",
				Parser:   "TestParser",
				Notifier: "TestNotifier",
			},
		},
	}, env)
	token := []byte{}
	req, _ := http.NewRequest("GET", "http://localhost/v1/glutton/save/token", nil)
	req.Header.Add(handler.TokenKeyHeader, "0123456789abcdef")
	testHTTPResponse(t, env.Server, req, func(w *httptest.ResponseRecorder) bool {
		assert.Equal(t, http.StatusOK, w.Code)
		token, _ = io.ReadAll(w.Body)
		return true
	})
	req, _ = http.NewRequest("POST", "http://localhost/v1/glutton/save", nil)
	req.Header.Add("token", string(token))
	testHTTPResponse(t, env.Server, req, func(w *httptest.ResponseRecorder) bool {
		assert.Equal(t, http.StatusOK, w.Code)
		p, _ := io.ReadAll(w.Body)
		log.Printf("server reply: %s", string(p))
		log.Printf("server header: %+v", w.HeaderMap)
		return true
	})
}

func testHTTPResponse(t *testing.T, r *gin.Engine, req *http.Request, f func(w *httptest.ResponseRecorder) bool) {
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if !f(w) {
		t.Fail()
	}
}
