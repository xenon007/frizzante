package frizzante

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gorilla/websocket"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

type PageFunction = func(req *Request, res *Response, p *Page)

type Server struct {
	hostName               string
	port                   int
	securePort             int
	multipartFormMaxMemory int64
	server                 *http.Server
	mux                    *http.ServeMux
	apiGuards              []func(req *Request, res *Response, pass func())
	pageGuards             []func(req *Request, res *Response, p *Page, pass func())
	sessions               map[string]*net.Conn
	readTimeout            time.Duration
	writeTimeout           time.Duration
	maxHeaderBytes         int
	certificate            string
	certificateKey         string
	notifier               *Notifier
	temporaryDirectory     string
	embeddedFileSystem     embed.FS
	webSocketUpgrader      *websocket.Upgrader
	sessionOperator        func(string) (
		get func(key string, defaultValue any) (value any),
		set func(key string, value any),
		unset func(key string),
		validate func() (valid bool),
		destroy func(),
	)
}

type sessionStore struct {
	createdAt      time.Time
	lastActivityAt time.Time
	data           map[string]any
}

// ServerCreate creates a server.
func ServerCreate() *Server {
	var memory = map[string]sessionStore{}

	return &Server{
		hostName:               "127.0.0.1",
		port:                   8081,
		securePort:             8383,
		multipartFormMaxMemory: 4096,
		server:                 nil,
		mux:                    http.NewServeMux(),
		sessions:               map[string]*net.Conn{},
		apiGuards:              []func(req *Request, res *Response, pass func()){},
		pageGuards:             []func(req *Request, res *Response, p *Page, pass func()){},
		readTimeout:            10 * time.Second,
		writeTimeout:           10 * time.Second,
		maxHeaderBytes:         3 * MB,
		certificate:            "",
		certificateKey:         "",
		temporaryDirectory:     ".temp",
		notifier:               NotifierCreate(),
		webSocketUpgrader: &websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
		sessionOperator: func(id string) (
			get func(key string, defaultValue any) (value any),
			set func(key string, value any),
			unset func(key string),
			validate func() (valid bool),
			destroy func(),
		) {
			store, exists := memory[id]
			if !exists {
				store = sessionStore{
					data:           map[string]any{},
					createdAt:      time.Now(),
					lastActivityAt: time.Now(),
				}
				memory[id] = store
			}

			get = func(key string, defaultValue any) (value any) {
				sessionItem, ok := store.data[key]
				if !ok {
					store.data[key] = defaultValue
					value = defaultValue
					store.lastActivityAt = time.Now()
					return
				}

				store.lastActivityAt = time.Now()
				value = sessionItem
				return
			}

			set = func(key string, value any) {
				store.lastActivityAt = time.Now()
				store.data[key] = value
			}

			unset = func(key string) {
				store.lastActivityAt = time.Now()
				delete(store.data, key)
			}

			validate = func() (valid bool) {
				elapsedSeconds := time.Since(store.lastActivityAt).Minutes()
				valid = elapsedSeconds < 30
				return
			}

			destroy = func() {
				delete(memory, id)
			}
			return
		},
	}
}

// ServerWithWebSocketReadBufferSize sets the maximum buffer size for each incoming web socket message.
// This will not limit the size of said messages.
func ServerWithWebSocketReadBufferSize(self *Server, readBufferSize int) {
	self.webSocketUpgrader.ReadBufferSize = readBufferSize
}

// ServerWithWebSocketWriteBufferSize sets the maximum buffer size for each outgoing web socket message.
// This will not limit the size of said messages.
func ServerWithWebSocketWriteBufferSize(self *Server, writeBufferSize int) {
	self.webSocketUpgrader.WriteBufferSize = writeBufferSize
}

// ServerWithMultipartFormMaxMemory sets the maximum memory for multipart forms before they fall back to disk.
func ServerWithMultipartFormMaxMemory(self *Server, multipartFormMaxMemory int64) {
	self.multipartFormMaxMemory = multipartFormMaxMemory
}

// ServerWithHostName sets the host name.
func ServerWithHostName(self *Server, hostName string) {
	self.hostName = hostName
}

// ServerWithPort sets the port.
func ServerWithPort(self *Server, port int) {
	self.port = port
}

// ServerWithSecurePort sets the secure port.
func ServerWithSecurePort(self *Server, securePort int) {
	self.securePort = securePort
}

// ServerWithReadTimeout sets the read timeout.
func ServerWithReadTimeout(self *Server, readTimeout time.Duration) {
	self.readTimeout = readTimeout
}

// ServerWithWriteTimeout sets the write timeout.
func ServerWithWriteTimeout(self *Server, writeTimeout time.Duration) {
	self.writeTimeout = writeTimeout
}

// ServerWithMaxHeaderBytes sets the maximum allowed bytes in the header of the request.
func ServerWithMaxHeaderBytes(self *Server, maxHeaderBytes int) {
	self.maxHeaderBytes = maxHeaderBytes
}

// ServerWithCertificateAndKey sets the tls configuration.
func ServerWithCertificateAndKey(self *Server, certificate string, key string) {
	self.certificate = certificate
	self.certificateKey = key
}

// ServerWithTemporaryDirectory sets the temporary directory.
func ServerWithTemporaryDirectory(self *Server, temporaryDirectory string) {
	self.temporaryDirectory = temporaryDirectory
}

// ServerWithEmbeddedFileSystem sets the embedded file system.
//
// The embedded file system should contain at least directory ".dist" so
// that the server can properly render and serve svelte components.
func ServerWithEmbeddedFileSystem(self *Server, embeddedFileSystem embed.FS) {
	self.embeddedFileSystem = embeddedFileSystem
}

// ServerWithNotifier sets the server notifier.
func ServerWithNotifier(self *Server, notifier *Notifier) {
	self.notifier = notifier
}

// ServerTemporaryFileSave sets a temporary file.
//
// When id is longer than 255 characters, the operation will fail silently and the server will be notified.
func ServerTemporaryFileSave(self *Server, id string, contents string) {
	if len(id) > 255 {
		NotifierSendError(self.notifier, fmt.Errorf("temporary file id is too long"))
		return
	}

	if strings.Contains(id, "../") {
		NotifierSendError(self.notifier, fmt.Errorf("invalid substring `../` detected in temporary file id `%s`", id))
		return
	}

	if !Exists(self.temporaryDirectory) {
		mkdirError := os.MkdirAll(self.temporaryDirectory, os.ModePerm)
		if mkdirError != nil {
			NotifierSendError(self.notifier, mkdirError)
			return
		}
	}

	fileName := self.temporaryDirectory
	if !strings.HasSuffix(fileName, "/") && !strings.HasPrefix(id, "/") {
		fileName += "/"
	}
	fileName += id

	directory := filepath.Dir(fileName)
	if !Exists(directory) {
		mkdirError := os.MkdirAll(directory, os.ModePerm)
		if mkdirError != nil {
			NotifierSendError(self.notifier, mkdirError)
			return
		}
	}

	var file *os.File

	if !ServerTemporaryFileExists(self, id) {
		fileLocal, createError := os.Create(fileName)
		if createError != nil {
			NotifierSendError(self.notifier, createError)
			return
		}
		file = fileLocal
	} else {
		fileLocal, openError := os.Open(fileName)
		if openError != nil {
			NotifierSendError(self.notifier, openError)
			return
		}
		file = fileLocal
	}

	_, writeError := file.WriteString(contents)
	if writeError != nil {
		NotifierSendError(self.notifier, writeError)
		return
	}

	closeError := file.Close()
	if closeError != nil {
		NotifierSendError(self.notifier, closeError)
		return
	}
}

// ServerTemporaryFile gets the contents o a temporary file.
func ServerTemporaryFile(self *Server, id string) string {
	if strings.Contains(id, "../") {
		NotifierSendError(self.notifier, fmt.Errorf("invalid substring `../` detected in temporary file id `%s`", id))
		return ""
	}

	fileName := self.temporaryDirectory
	if !strings.HasSuffix(fileName, "/") && !strings.HasPrefix(id, "/") {
		fileName += "/"
	}
	fileName += id
	contents, err := os.ReadFile(fileName)
	if err != nil {
		NotifierSendError(self.notifier, err)
		return ""
	}
	return string(contents)
}

// ServerTemporaryFileExists checks if a temporary file Exists.
//
// When id is longer than 255 characters, the operation will fail silently and the server will be notified.
func ServerTemporaryFileExists(self *Server, id string) bool {
	if len(id) > 255 {
		NotifierSendError(self.notifier, fmt.Errorf("temporary file id is too long"))
		return false
	}

	if strings.Contains(id, "../") {
		return false
	}

	fileName := self.temporaryDirectory
	if !strings.HasSuffix(fileName, "/") && !strings.HasPrefix(id, "/") {
		fileName += "/"
	}
	fileName += id
	return Exists(fileName)
}

// ServerTemporaryDirectoryClear clears the temporary directory.
func ServerTemporaryDirectoryClear(self *Server) {
	err := os.RemoveAll(self.temporaryDirectory)
	if err != nil {
		NotifierSendError(self.notifier, err)
	}
}

// ReceiveCookie reads the contents of a cookie from the message and returns the value.
//
// Compatible with web sockets.
func ReceiveCookie(self *Request, key string) string {
	cookie, cookieError := self.httpRequest.Cookie(key)
	if cookieError != nil {
		NotifierSendError(self.server.notifier, cookieError)
		return ""
	}
	value, unescapeError := url.QueryUnescape(cookie.Value)
	if unescapeError != nil {
		return ""
	}

	return value
}

// ReceiveMessage reads the contents of the message and returns the value.
//
// Compatible with web sockets.
func ReceiveMessage(self *Request) string {
	if self.webSocketConn != nil {
		_, readBytes, readError := self.webSocketConn.ReadMessage()
		if readError != nil {
			NotifierSendError(self.server.notifier, readError)
			return ""
		}
		return string(readBytes)
	}

	readBytes, readAllError := io.ReadAll(self.httpRequest.Body)
	if readAllError != nil {
		NotifierSendError(self.server.notifier, readAllError)
		return ""
	}
	return string(readBytes)
}

// ReceiveJson reads the message as json and stores the result into value.
//
// Compatible with web sockets.
func ReceiveJson[T any](self *Request) (*T, bool) {
	var value T
	if self.webSocketConn != nil {
		jsonError := self.webSocketConn.ReadJSON(value)
		if jsonError != nil {
			NotifierSendError(self.server.notifier, jsonError)
			return nil, false
		}
		return &value, true
	}

	readBytes, readAllError := io.ReadAll(self.httpRequest.Body)
	if readAllError != nil {
		NotifierSendError(self.server.notifier, readAllError)
		return nil, false
	}
	unmarshalError := json.Unmarshal(readBytes, &value)
	if unmarshalError != nil {
		NotifierSendError(self.server.notifier, unmarshalError)
		return nil, false
	}
	return &value, true
}

// ReceiveForm reads the message as a form and returns the value.
func ReceiveForm(self *Request) *url.Values {
	if self.webSocketConn != nil {
		NotifierSendError(self.server.notifier, errors.New("web socket connections cannot receive form payloads"))
		return &url.Values{}
	}

	parseMultipartFormError := self.httpRequest.ParseMultipartForm(self.server.multipartFormMaxMemory)
	if parseMultipartFormError != nil {
		if !errors.Is(parseMultipartFormError, http.ErrNotMultipart) {
			NotifierSendError(self.server.notifier, parseMultipartFormError)
		}

		parseFormError := self.httpRequest.ParseForm()
		if parseFormError != nil {
			NotifierSendError(self.server.notifier, parseFormError)
		}
	}

	return &self.httpRequest.Form
}

// ReceiveQuery reads a query field and returns the value.
//
// Compatible with web sockets.
func ReceiveQuery(self *Request, name string) string {
	return self.httpRequest.URL.Query().Get(name)
}

// ReceivePath reads a parameters fields and returns the value.
//
// Compatible with web sockets.
func ReceivePath(self *Request, name string) string {
	return self.httpRequest.PathValue(name)
}

// ReceiveHeader reads a header field and returns the value.
//
// Compatible with web sockets.
func ReceiveHeader(self *Request, key string) string {
	return self.httpRequest.Header.Get(key)
}

// ReceiveContentType reads the Content-Type header field and returns the value.
//
// Compatible with web sockets.
func ReceiveContentType(self *Request) string {
	return self.httpRequest.Header.Get("Content-Type")
}

// ServerStart starts the server.
//
// If the server fails to start, ServerStart panics.
func ServerStart(self *Server) {
	logger := log.New(self.notifier.errorFile, "<error>", log.Ltime|log.Llongfile)

	self.server = &http.Server{
		Handler:        self.mux,
		ReadTimeout:    self.readTimeout,
		WriteTimeout:   self.writeTimeout,
		MaxHeaderBytes: self.maxHeaderBytes,
		ErrorLog:       logger,
	}

	if !entryCreated {
		ServerWithApi(self, "GET /",
			func(request *Request, response *Response) {
				SendStatus(response, 404)
			},
		)
	}

	var waiter sync.WaitGroup

	waiter.Add(2)

	go func() {
		address := fmt.Sprintf("%s:%d", self.hostName, self.port)
		NotifierSendMessage(self.notifier, fmt.Sprintf("listening for requests at http://%s", address))
		err := http.ListenAndServe(address, self.mux)
		if err != nil {
			if errors.Is(err, http.ErrServerClosed) {
				NotifierSendMessage(self.notifier, "shutting down server")
				return
			}
			panic(err.Error())
		}
	}()

	go func() {
		secureAddress := fmt.Sprintf("%s:%d", self.hostName, self.securePort)
		if "" != self.certificate && "" != self.certificateKey {
			NotifierSendMessage(self.notifier, fmt.Sprintf("listening for requests at https://%s", secureAddress))
			err := http.ListenAndServeTLS(secureAddress, self.certificate, self.certificateKey, self.mux)
			if err != nil {
				if errors.Is(err, http.ErrServerClosed) {
					NotifierSendMessage(self.notifier, "shutting down server")
					return
				}
				panic(err.Error())
			}
		}
	}()

	waiter.Wait()
}

// ServerStop attempts to stop the server.
//
// If the shutdown attempt fails, ServerStop panics.
func ServerStop(self *Server) {
	err := self.server.Shutdown(context.Background())
	if err != nil {
		panic(err.Error())
	}
}

var pathParametersPattern = regexp.MustCompile(`{([^{}]+)}`)

type Route struct {
	server   *Server
	isPage   bool
	page     string
	callback func(request *Request, response *Response)
	mount    func(pattern string)
}

// routeCreate creates a route configuration from a callback function.
func routeCreate(
	callback func(
		request *Request,
		response *Response,
	),
) *Route {
	return &Route{
		isPage: false,
		page:   "",
		callback: func(request *Request, response *Response) {
			for _, guard := range response.server.apiGuards {
				pass := false
				guard(request, response, func() {
					pass = true
				})

				if !pass {
					return
				}
			}

			callback(request, response)
		},
		mount: func(pattern string) {},
	}
}

// routeCreateWithPage creates a route configuration from a callback function, just like routeCreate.
//
// Unlike routeCreate, routeCreateWithPage also creates a Page, which is used to automatically
// to serve a svelte page after invoking callback.
//
// Generally speaking, you should never manually invoke SendEcho or similar functions.
//
// However, it is safe to invoke receive functions, like ReceiveHeader, ReceiveCookie, etc.
func routeCreateWithPage(
	page string,
	callback PageFunction,
) *Route {
	var pattern string

	return &Route{
		isPage: true,
		page:   page,
		callback: func(
			request *Request,
			response *Response,
		) {
			p := &Page{
				render:     RenderFull,
				data:       map[string]any{},
				efs:        request.server.embeddedFileSystem,
				name:       page,
				parameters: map[string]string{},
			}

			for _, guard := range response.server.pageGuards {
				pass := false
				guard(request, response, p, func() {
					pass = true
				})

				if !pass {
					return
				}
			}

			callback(request, response, p)

			if nil != response.navigate {
				SendRedirect(response, response.navigate.Location, http.StatusFound)
				return
			}

			if "" != response.header.Get("Location") {
				return
			}

			if nil == p {
				NotifierSendError(request.server.notifier, fmt.Errorf("svelte page handler `%s` returned a nil page", pattern))
				return
			}

			if nil == p.data {
				p.data = map[string]any{}
			}

			if VerifyAccept(request, "application/json") {
				data, marshalError := json.Marshal(p.data)
				if marshalError != nil {
					NotifierSendError(request.server.notifier, marshalError)
					return
				}
				SendHeader(response, "Content-Type", "application/json")
				SendEcho(response, string(data))
				return
			}

			if nil == p.parameters {
				p.parameters = map[string]string{}
			}

			for _, name := range pathParametersPattern.FindAllStringSubmatch(pattern, -1) {
				if len(name) < 1 {
					continue
				}
				p.parameters[name[1]] = request.httpRequest.PathValue(name[1])
			}

			SendPage(response, p)
		},
		mount: func(patternLocal string) {
			pattern = patternLocal
			patternParts := strings.Split(patternLocal, " ")
			patternCounter := len(patternParts)
			if patternCounter > 1 {
				pages[page] = path.Join(patternParts[1:]...)
			}
		},
	}
}

var entryCreated = false

// serverMap maps a pattern to a given route.
//
// If the given pattern conflicts with one that is already registered, serverMap panics.
func serverMap(
	self *Server,
	pattern string,
	route *Route,
) {
	patternParts := strings.Split(pattern, " ")
	patternCounter := len(patternParts)
	isEntry := patternCounter > 1 && strings.HasPrefix(strings.TrimPrefix(filepath.Join(patternParts[1:]...), " "), "/")

	if isEntry && !entryCreated {
		entryCreated = true
	}

	if route.mount != nil {
		route.mount(pattern)
	}

	self.mux.HandleFunc(pattern, func(writer http.ResponseWriter, httpRequest *http.Request) {
		request := Request{
			server:      self,
			httpRequest: httpRequest,
		}

		httpHeader := writer.Header()

		response := Response{
			server:                self,
			writer:                &writer,
			lockedStatusAndHeader: false,
			statusCode:            200,
			header:                &httpHeader,
			eventName:             "",
			eventId:               1,
		}

		request.response = &response
		response.request = &request

		if isEntry {
			SendEmbeddedFileOrElse(&response, func() {
				SendFileOrElse(&response, func() {
					if route.callback != nil {
						if "/favicon.ico" == request.httpRequest.RequestURI {
							SendNotFound(&response)
							return
						}
						route.callback(&request, &response)

						if !response.lockedStatusAndHeader {
							SendEcho(&response, "")
						}
					}
				})
			})
		} else if route.callback != nil {
			route.callback(&request, &response)

			if !response.lockedStatusAndHeader {
				SendEcho(&response, "")
			}
		}
	})
}

type Request struct {
	server        *Server
	response      *Response
	httpRequest   *http.Request
	webSocketConn *websocket.Conn
}

type Navigate struct {
	Page       string
	Parameters map[string]string
	Location   string
}

type Response struct {
	server                *Server
	request               *Request
	writer                *http.ResponseWriter
	lockedStatusAndHeader bool
	statusCode            int
	header                *http.Header
	webSocket             *websocket.Conn
	eventName             string
	navigate              *Navigate
	eventId               int64
}

// SendNavigateWithParameters sends the client an instruction to navigate.
func SendNavigateWithParameters(self *Response, page string, parameters map[string]string) {
	if nil == parameters {
		parameters = map[string]string{}
	}

	p, pathFound := pages[page]
	if !pathFound {
		NotifierSendError(self.server.notifier, fmt.Errorf("redirect to page `%s` failed because page id `%s` is unknown", page, page))
		return
	}

	location := string(
		pathFieldRegex.ReplaceAllFunc(
			[]byte(p),
			func(i []byte) []byte {
				if nil == parameters {
					return []byte{}
				}
				key := string(i[1 : len(i)-1])
				return []byte(parameters[key])
			},
		),
	)

	self.navigate = &Navigate{
		Page:       page,
		Parameters: parameters,
		Location:   location,
	}
}

// SendNavigate sends the client an instruction to navigate.
func SendNavigate(self *Response, page string) {
	SendNavigateWithParameters(self, page, map[string]string{})
}

// SendRedirect redirects the request.
func SendRedirect(self *Response, location string, statusCode int) {
	SendStatus(self, statusCode)
	SendHeader(self, "Location", location)
}

var pathFieldRegex = regexp.MustCompile(`\{(.*?)\}`)

// SendRedirectToSecure tries to redirect the request to the https server.
//
// When the request is already secure, SendRedirectToSecure returns false.
func SendRedirectToSecure(self *Response, statusCode int) bool {
	request := self.request
	if "" == request.server.certificate || "" == request.server.certificateKey || request.httpRequest.TLS != nil {
		return false
	}

	insecureSuffix := fmt.Sprintf(":%d", request.server.port)
	secureSuffix := fmt.Sprintf(":%d", request.server.securePort)
	secureHost := strings.Replace(request.httpRequest.Host, insecureSuffix, secureSuffix, 1)
	secureLocation := fmt.Sprintf("https://%s%s", secureHost, request.httpRequest.RequestURI)
	SendRedirect(self, secureLocation, 302)
	return true
}

// SendStatus sets the status code.
//
// This will lock the status, which makes it
// so that the next time you invoke this
// function it will fail with an error.
//
// You can retrieve the error using ServerRecallError.
func SendStatus(self *Response, code int) {
	if self.lockedStatusAndHeader {
		NotifierSendError(self.server.notifier, errors.New("status is locked"))
		return
	}
	self.statusCode = code
}

// SendHeader sets a header field.
//
// If the status has not been sent already, a default "200 OK" status will be sent immediately.
//
// This means the status will become locked and further attempts to send the status will fail with an error.
//
// You can retrieve the error using ServerRecallError
func SendHeader(self *Response, key string, value string) {
	if self.lockedStatusAndHeader {
		NotifierSendError(self.server.notifier, errors.New("headers locked"))
		return
	}

	self.header.Set(key, value)
}

// SendContentType sets the Content-Type header field.
func SendContentType(self *Response, contentType string) {
	SendHeader(self, "Content-Type", contentType)
}

// SendCookie sends a cookies to the client.
func SendCookie(self *Response, key string, value string) {
	SendHeader(self, "Set-Cookie", fmt.Sprintf("%s=%s; Path=/; HttpOnly", url.QueryEscape(key), url.QueryEscape(value)))
}

// SendContent sends binary safe content.
//
// If the status code or the header have not been sent already, a default status of "200 OK" will be sent immediately along with whatever headers you've previously defined.
//
// The status code and the header will become locked and further attempts to send either of them will fail with an error.
//
// You can retrieve the error using ServerRecallError.
//
// Compatible with web sockets.
func SendContent(self *Response, content []byte) {
	if !self.lockedStatusAndHeader {
		(*self.writer).WriteHeader(self.statusCode)
		self.lockedStatusAndHeader = true
	}

	if self.webSocket != nil {
		writeError := self.webSocket.WriteMessage(websocket.TextMessage, content)
		if writeError != nil {
			NotifierSendError(self.server.notifier, writeError)
		}
		return
	}

	_, err := (*self.writer).Write(content)
	if err != nil {
		NotifierSendError(self.server.notifier, err)
		return
	}
}

// SendEcho sends utf-8 safe content.
//
// If the status code or the header have not been sent already, a default status of "200 OK" will be sent immediately along with whatever headers you've previously defined.
//
// The status code and the header will become locked and further attempts to send either of them will fail with an error.
//
// You can retrieve the error using ServerRecallError.
//
// Compatible with web sockets.
func SendEcho(self *Response, content string) {
	SendContent(self, []byte(content))
}

// SendNotFound sends an empty echo with status 404 Not Found.
func SendNotFound(self *Response) {
	SendStatus(self, http.StatusNotFound)
}

// SendUnauthorized sends an empty echo with status 401 Unauthorized.
func SendUnauthorized(self *Response) {
	SendStatus(self, http.StatusUnauthorized)
}

// SendBadRequest sends an empty echo with status 400 Bad Request.
func SendBadRequest(self *Response) {
	SendStatus(self, http.StatusBadRequest)
}

// SendForbidden sends an empty echo with status 403 Forbidden.
func SendForbidden(self *Response) {
	SendStatus(self, http.StatusForbidden)
}

// SendTooManyRequests sends and empty echo with status 403 Forbidden.
func SendTooManyRequests(self *Response) {
	SendStatus(self, http.StatusTooManyRequests)
}

// SendJson sends json content.
//
// If the status code or the header have not been sent already, a default status of "200 OK" will be sent immediately along with whatever headers you've previously defined.
//
// The status code and the header will become locked and further attempts to send either of them will fail with an error.
//
// You can retrieve the error using ServerRecallError.
//
// Compatible with web sockets.
func SendJson(self *Response, payload any) {
	content, marshalError := json.Marshal(payload)
	if marshalError != nil {
		NotifierSendError(self.server.notifier, marshalError)
	}

	if nil == self.webSocket {
		contentType := self.header.Get("Content-Type")
		if "" == contentType {
			self.header.Set("Content-Type", "application/json")
		}
	}

	SendContent(self, content)
}

// VerifyContentType checks if the incoming request has any of the given content-types.
func VerifyContentType(self *Request, contentTypes ...string) bool {
	requestedMime := self.httpRequest.Header.Get("Content-Type")
	for _, acceptedMime := range contentTypes {
		if acceptedMime == "*" || strings.HasPrefix(requestedMime, acceptedMime) {
			return true
		}
	}

	return false
}

// VerifyAccept checks if the incoming request accepts any of the given content-types.
func VerifyAccept(self *Request, contentTypes ...string) bool {
	requestedAcceptMime := self.httpRequest.Header.Get("Accept")
	for _, acceptedMime := range contentTypes {
		if acceptedMime == "*" || strings.Contains(requestedAcceptMime, acceptedMime) {
			return true
		}
	}

	return false
}

func sendEventContent(self *Response, content []byte) {
	header := fmt.Sprintf("id: %d\r\nevent: %s\r\n", self.eventId, self.eventName)

	_, writeEventError := (*self.writer).Write([]byte(header))
	if writeEventError != nil {
		NotifierSendError(self.server.notifier, writeEventError)
		return
	}

	for _, line := range bytes.Split(content, []byte("\r\n")) {
		_, writeEventError = (*self.writer).Write([]byte("data: "))
		if writeEventError != nil {
			NotifierSendError(self.server.notifier, writeEventError)
			return
		}

		_, writeEventError = (*self.writer).Write(line)
		if writeEventError != nil {
			NotifierSendError(self.server.notifier, writeEventError)
			return
		}

		_, writeEventError = (*self.writer).Write([]byte("\r\n"))
		if writeEventError != nil {
			NotifierSendError(self.server.notifier, writeEventError)
			return
		}
	}

	_, writeEventError = (*self.writer).Write([]byte("\r\n"))
	if writeEventError != nil {
		NotifierSendError(self.server.notifier, writeEventError)
		return
	}

	flusher, flushedOk := (*self.writer).(http.Flusher)
	if !flushedOk {
		NotifierSendError(self.server.notifier, errors.New("could not retrieve flusher"))
		return
	}

	flusher.Flush()

	self.eventId++
}

// SendEmbeddedFileOrIndexOrElse sends the embedded file requested by the client,
// or the closest index.html embedded file, or else falls back.
func SendEmbeddedFileOrIndexOrElse(self *Response, orElse func()) {
	request := self.request
	fileName := filepath.Join(".dist", "client", request.httpRequest.RequestURI)

	if !EmbeddedExists(request.server.embeddedFileSystem, fileName) {
		orElse()
		return
	}

	if EmbeddedIsDirectory(request.server.embeddedFileSystem, fileName) {
		fileName = filepath.Join(fileName, "index.html")
		if !IsFile(fileName) {
			orElse()
			return
		}
	}

	reader, info, readerError := createReaderFromEmbeddedFileName(&request.server.embeddedFileSystem, fileName)
	if readerError != nil {
		NotifierSendError(self.server.notifier, readerError)
		return
	}

	if self.webSocket != nil {
		content, readError := io.ReadAll(reader)
		if readError != nil {
			NotifierSendError(self.server.notifier, readError)
			return
		}
		writeError := self.webSocket.WriteMessage(websocket.TextMessage, content)
		if writeError != nil {
			NotifierSendError(self.server.notifier, writeError)
		}
		return
	}

	if "" != self.eventName {
		content, readError := io.ReadAll(reader)
		if readError != nil {
			NotifierSendError(self.server.notifier, readError)
			return
		}
		sendEventContent(self, content)
		return
	}

	if "" == self.header.Get("Content-Type") {
		SendHeader(self, "Content-Type", Mime(fileName))
	}

	if "" == self.header.Get("Content-Length") {
		SendHeader(self, "Content-Length", fmt.Sprintf("%d", (*info).Size()))
	}
	http.ServeContent(*self.writer, request.httpRequest, fileName, (*info).ModTime(), reader)
}

// SendEmbeddedFileOrElse sends the embedded file requested by the client,
// or the closest index.html embedded file, or else falls back.
func SendEmbeddedFileOrElse(self *Response, orElse func()) {
	request := self.request
	fileName := filepath.Join(".dist", "client", request.httpRequest.RequestURI)
	fileName = strings.Split(fileName, "?")[0]
	fileName = strings.Split(fileName, "&")[0]

	if !EmbeddedExists(request.server.embeddedFileSystem, fileName) ||
		EmbeddedIsDirectory(request.server.embeddedFileSystem, fileName) {
		orElse()
		return
	}

	reader, info, readerError := createReaderFromEmbeddedFileName(&request.server.embeddedFileSystem, fileName)
	if readerError != nil {
		NotifierSendError(self.server.notifier, readerError)
		return
	}

	if self.webSocket != nil {
		content, readError := io.ReadAll(reader)
		if readError != nil {
			NotifierSendError(self.server.notifier, readError)
			return
		}
		writeError := self.webSocket.WriteMessage(websocket.TextMessage, content)
		if writeError != nil {
			NotifierSendError(self.server.notifier, writeError)
		}
		return
	}

	if "" != self.eventName {
		content, readError := io.ReadAll(reader)
		if readError != nil {
			NotifierSendError(self.server.notifier, readError)
			return
		}
		sendEventContent(self, content)
		return
	}

	if "" == self.header.Get("Content-Type") {
		SendHeader(self, "Content-Type", Mime(fileName))
	}

	if "" == self.header.Get("Content-Length") {
		SendHeader(self, "Content-Length", fmt.Sprintf("%d", (*info).Size()))
	}
	http.ServeContent(*self.writer, request.httpRequest, fileName, (*info).ModTime(), reader)
}

// SendFileOrIndexOrElse sends the file requested by the client,
// or the closest index.html file, or else falls back.
func SendFileOrIndexOrElse(self *Response, orElse func()) {
	request := self.request
	fileName := filepath.Join(".dist", "client", request.httpRequest.RequestURI)

	if !Exists(fileName) {
		orElse()
		return
	}

	if IsDirectory(fileName) {
		fileName = filepath.Join(fileName, "index.html")
		if !IsFile(fileName) {
			orElse()
			return
		}
	}

	reader, info, readerError := createReaderFromFileName(fileName)
	if readerError != nil {
		NotifierSendError(self.server.notifier, readerError)
		return
	}

	if self.webSocket != nil {
		content, readError := io.ReadAll(reader)
		if readError != nil {
			NotifierSendError(self.server.notifier, readError)
			return
		}
		writeError := self.webSocket.WriteMessage(websocket.TextMessage, content)
		if writeError != nil {
			NotifierSendError(self.server.notifier, writeError)
		}
		return
	}

	if "" != self.eventName {
		content, readError := io.ReadAll(reader)
		if readError != nil {
			NotifierSendError(self.server.notifier, readError)
			return
		}
		sendEventContent(self, content)
		return
	}

	if "" == self.header.Get("Content-Type") {
		SendHeader(self, "Content-Type", Mime(fileName))
	}

	if "" == self.header.Get("Content-Length") {
		SendHeader(self, "Content-Length", fmt.Sprintf("%d", (*info).Size()))
	}
	http.ServeContent(*self.writer, request.httpRequest, fileName, (*info).ModTime(), reader)
}

// SendFileOrElse sends the file requested by the client, or else falls back.
func SendFileOrElse(self *Response, orElse func()) {
	request := self.request
	fileName := filepath.Join(".dist", "client", request.httpRequest.RequestURI)

	if !Exists(fileName) || IsDirectory(fileName) {
		orElse()
		return
	}

	reader, info, readerError := createReaderFromFileName(fileName)
	if readerError != nil {
		NotifierSendError(self.server.notifier, readerError)
		return
	}

	if self.webSocket != nil {
		content, readError := io.ReadAll(reader)
		if readError != nil {
			NotifierSendError(self.server.notifier, readError)
			return
		}
		writeError := self.webSocket.WriteMessage(websocket.TextMessage, content)
		if writeError != nil {
			NotifierSendError(self.server.notifier, writeError)
		}
		return
	}

	if "" != self.eventName {
		content, readError := io.ReadAll(reader)
		if readError != nil {
			NotifierSendError(self.server.notifier, readError)
			return
		}
		sendEventContent(self, content)
		return
	}

	if "" == self.header.Get("Content-Type") {
		SendHeader(self, "Content-Type", Mime(fileName))
	}

	if "" == self.header.Get("Content-Length") {
		SendHeader(self, "Content-Length", fmt.Sprintf("%d", (*info).Size()))
	}
	http.ServeContent(*self.writer, request.httpRequest, fileName, (*info).ModTime(), reader)
}

func createReaderFromEmbeddedFileName(efs *embed.FS, fileName string) (*bytes.Reader, *os.FileInfo, error) {
	file, openError := efs.Open(fileName)
	if openError != nil {
		return nil, nil, openError
	}

	defer file.Close()
	fileInfo, _ := file.Stat()

	buffer := make([]byte, fileInfo.Size())
	_, readError := file.Read(buffer)
	if readError != nil {
		return nil, nil, readError
	}

	return bytes.NewReader(buffer), &fileInfo, nil
}

func createReaderFromFileName(fileName string) (*bytes.Reader, *os.FileInfo, error) {
	file, openError := os.Open(fileName)
	if openError != nil {
		return nil, nil, openError
	}

	defer file.Close()
	fileInfo, _ := file.Stat()

	buffer := make([]byte, fileInfo.Size())
	_, readError := file.Read(buffer)
	if readError != nil {
		return nil, nil, readError
	}

	return bytes.NewReader(buffer), &fileInfo, nil
}

// SendWebSocketUpgrade upgrades the http connection to web socket.
func SendWebSocketUpgrade(self *Response, callback func()) {
	request := self.request
	conn, upgradeError := self.server.webSocketUpgrader.Upgrade(*self.writer, request.httpRequest, nil)
	if upgradeError != nil {
		NotifierSendError(request.server.notifier, upgradeError)
		return
	}
	defer conn.Close()
	self.webSocket = conn
	request.webSocketConn = conn
	self.lockedStatusAndHeader = true
	callback()
}

// SendPage renders and echos a svelte page.
func SendPage(self *Response, page *Page) {
	content, compileError := PageCompile(page)
	if nil != compileError {
		NotifierSendError(self.server.notifier, compileError)
		return
	}

	if "" == self.header.Get("Content-Type") {
		SendHeader(self, "Content-Type", "text/html")
	}

	SendEcho(self, content)
}

// ServerWithSessionOperator sets the session operator,
// which is a function that provides the four main
// operations used by the server to manage any session,
// get, set, unset and destroy.
//
// Get must retriedataata from the session store.
//
// Set must create a new property to the session store or update an existing one.
//
// Unset must remove a property from the session store.
//
// Destroy must destroy the whole session, store included.
//
// In this context, "store", is any type dataata storage,
// it could be a file written to disk, a database, Ram,
// it doesn't matter.
//
// The only thing that matters is a consistent
// implementation of the four operations.
func ServerWithSessionOperator(
	self *Server,
	sessionOperator func(string) (
		get func(key string, defaultValue any) (value any),
		set func(key string, value any),
		unset func(key string),
		validate func() (valid bool),
		destroy func(),
	),
) {
	self.sessionOperator = sessionOperator
}

// ServerWithApi adds an api.
func ServerWithApi(
	self *Server,
	pattern string,
	callback func(
		request *Request,
		response *Response,
	),
) {
	serverMap(self, pattern, routeCreate(callback))
}

type ApiGuardFunction = func(req *Request, res *Response, pass func())

// ServerWithApiGuard adds an api guard, a function that executes before every api request.
func ServerWithApiGuard(self *Server, guard ApiGuardFunction) {
	self.apiGuards = append(self.apiGuards, guard)
}

type WireFunction = func()
type LoadFunction = func(wire WireFunction)
type IndexFunction = func(
	route func(path string, page string),
	show func(showFunction PageFunction),
	action func(actionFunction PageFunction),
)

// ServerWithIndex adds an index.
func ServerWithIndex(
	self *Server,
	index IndexFunction,
) {
	indexPage := ""
	indexPath := ""
	var show PageFunction
	var action PageFunction

	index(
		func(path string, page string) {
			indexPath = path
			indexPage = page
		},
		func(showFunction PageFunction) {
			show = showFunction
		},
		func(actionFunction PageFunction) {
			action = actionFunction
		},
	)

	if "" == indexPath {
		indexPath = "/" + strings.ReplaceAll(indexPage, ".", "/")
	}

	if "" == indexPage {
		NotifierSendError(self.notifier, fmt.Errorf("could not add index because page `%s` is unknown", indexPage))
		return
	}

	if "" == indexPath {
		NotifierSendError(self.notifier, fmt.Errorf("could not add index because path `%s` is unknown", indexPath))
		return
	}

	if nil == show {
		show = func(req *Request, res *Response, p *Page) {
			// Noop.
		}
	}

	if nil == action {
		action = func(req *Request, res *Response, p *Page) {
			// Noop.
		}
	}

	serverMap(self, "GET "+indexPath, routeCreateWithPage(indexPage, show))
	serverMap(self, "POST "+indexPath, routeCreateWithPage(indexPage, action))
}

type PageGuardFunction = func(req *Request, res *Response, p *Page, pass func())

// ServerWithPageGuard adds a page guard, a function that executes before every api page.
func ServerWithPageGuard(self *Server, guard PageGuardFunction) {
	self.pageGuards = append(self.pageGuards, guard)
}
