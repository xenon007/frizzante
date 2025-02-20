package frizzante

import (
	"bytes"
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gorilla/websocket"
	uuid "github.com/nu7hatch/gouuid"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"sync"
	"time"
)

type Server struct {
	hostName               string
	port                   int
	securePort             int
	multipartFormMaxMemory int64
	server                 *http.Server
	database               *sql.DB
	mux                    *http.ServeMux
	sessions               map[string]*net.Conn
	readTimeout            time.Duration
	writeTimeout           time.Duration
	maxHeaderBytes         int
	logger                 *log.Logger
	certificate            string
	certificateKey         string
	errorHandler           func(error)
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
		hostName:           "127.0.0.1",
		port:               8081,
		securePort:         8383,
		server:             nil,
		mux:                http.NewServeMux(),
		sessions:           map[string]*net.Conn{},
		readTimeout:        10 * time.Second,
		writeTimeout:       10 * time.Second,
		maxHeaderBytes:     3 * MB,
		logger:             log.Default(),
		certificate:        "",
		certificateKey:     "",
		errorHandler:       func(error) {},
		temporaryDirectory: ".temp",
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
					value = store.data[key]
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

// ServerWithDatabase sets the server database.
func ServerWithDatabase(self *Server, database *sql.DB) {
	self.database = database
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

// ServerWithLogger sets the server logger.
func ServerWithLogger(self *Server, logger *log.Logger) {
	self.logger = logger
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

// ServerTemporaryFileSave sets a temporary file.
//
// When id is longer than 255 characters, the operation will fail silently and the server will be notified.
func ServerTemporaryFileSave(self *Server, id string, contents string) {
	if len(id) > 255 {
		ServerNotifyError(self, fmt.Errorf("temporary file id is too long"))
		return
	}

	if strings.Contains(id, "../") {
		ServerNotifyError(self, fmt.Errorf("invalid substring `../` detected in temporary file id `%s`", id))
		return
	}

	if !Exists(self.temporaryDirectory) {
		mkdirError := os.MkdirAll(self.temporaryDirectory, os.ModePerm)
		if mkdirError != nil {
			ServerNotifyError(self, mkdirError)
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
			ServerNotifyError(self, mkdirError)
			return
		}
	}

	var file *os.File

	if !ServerTemporaryFileExists(self, id) {
		fileLocal, createError := os.Create(fileName)
		if createError != nil {
			ServerNotifyError(self, createError)
			return
		}
		file = fileLocal
	} else {
		fileLocal, openError := os.Open(fileName)
		if openError != nil {
			ServerNotifyError(self, openError)
			return
		}
		file = fileLocal
	}

	_, writeError := file.WriteString(contents)
	if writeError != nil {
		ServerNotifyError(self, writeError)
		return
	}

	closeError := file.Close()
	if closeError != nil {
		ServerNotifyError(self, closeError)
		return
	}
}

// ServerTemporaryFile gets the contents o a temporary file.
func ServerTemporaryFile(self *Server, id string) string {
	if strings.Contains(id, "../") {
		ServerNotifyError(self, fmt.Errorf("invalid substring `../` detected in temporary file id `%s`", id))
		return ""
	}

	fileName := self.temporaryDirectory
	if !strings.HasSuffix(fileName, "/") && !strings.HasPrefix(id, "/") {
		fileName += "/"
	}
	fileName += id
	contents, err := os.ReadFile(fileName)
	if err != nil {
		ServerNotifyError(self, err)
		return ""
	}
	return string(contents)
}

// ServerTemporaryFileExists checks if a temporary file Exists.
//
// When id is longer than 255 characters, the operation will fail silently and the server will be notified.
func ServerTemporaryFileExists(self *Server, id string) bool {
	if len(id) > 255 {
		ServerNotifyError(self, fmt.Errorf("temporary file id is too long"))
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
		ServerNotifyError(self, err)
	}
}

// ReceiveCookie reads the contents of a cookie from the message and returns the value.
//
// Compatible with web sockets.
func ReceiveCookie(self *Request, key string) string {
	cookie, cookieError := self.HttpRequest.Cookie(key)
	if cookieError != nil {
		ServerNotifyError(self.server, cookieError)
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
	if self.webSocket != nil {
		_, readBytes, readError := self.webSocket.ReadMessage()
		if readError != nil {
			ServerNotifyError(self.server, readError)
			return ""
		}
		return string(readBytes)
	}

	readBytes, readAllError := io.ReadAll(self.HttpRequest.Body)
	if readAllError != nil {
		ServerNotifyError(self.server, readAllError)
		return ""
	}
	return string(readBytes)
}

// ReceiveJson reads the message as json and stores the result into value.
//
// Compatible with web sockets.
func ReceiveJson[T any](self *Request, value *T) {
	if self.webSocket != nil {
		jsonError := self.webSocket.ReadJSON(value)
		if jsonError != nil {
			ServerNotifyError(self.server, jsonError)
			return
		}
		return
	}

	readBytes, readAllError := io.ReadAll(self.HttpRequest.Body)
	if readAllError != nil {
		ServerNotifyError(self.server, readAllError)
		return
	}
	unmarshalError := json.Unmarshal(readBytes, &value)
	if unmarshalError != nil {
		ServerNotifyError(self.server, unmarshalError)
	}
}

// ReceiveForm reads the message as a form and returns the value.
func ReceiveForm(self *Request) *url.Values {
	if self.webSocket != nil {
		ServerNotifyError(self.server, errors.New("web socket connections cannot receive form payloads"))
		return nil
	}

	parseMultipartFormError := self.HttpRequest.ParseMultipartForm(self.server.multipartFormMaxMemory)
	if parseMultipartFormError != nil {
		if !errors.Is(parseMultipartFormError, http.ErrNotMultipart) {
			ServerNotifyError(self.server, parseMultipartFormError)
		}

		parseFormError := self.HttpRequest.ParseForm()
		if parseFormError != nil {
			ServerNotifyError(self.server, parseFormError)
		}
	}

	return &self.HttpRequest.Form
}

// ReceiveQuery reads a query field and returns the value.
//
// Compatible with web sockets.
func ReceiveQuery(self *Request, name string) string {
	return self.HttpRequest.URL.Query().Get(name)
}

// ReceivePath reads a path fields and returns the value.
//
// Compatible with web sockets.
func ReceivePath(self *Request, name string) string {
	return self.HttpRequest.PathValue(name)
}

// ReceiveHeader reads a header field and returns the value.
//
// Compatible with web sockets.
func ReceiveHeader(self *Request, key string) string {
	return self.HttpRequest.Header.Get(key)
}

// ReceiveContentType reads the Content-Type header field and returns the value.
//
// Compatible with web sockets.
func ReceiveContentType(self *Request) string {
	return self.HttpRequest.Header.Get("Content-Type")
}

// ServerStart starts the server.
//
// If the server fails to start, ServerStart panics.
func ServerStart(self *Server) {
	self.server = &http.Server{
		Handler:        self.mux,
		ReadTimeout:    self.readTimeout,
		WriteTimeout:   self.writeTimeout,
		MaxHeaderBytes: self.maxHeaderBytes,
		ErrorLog:       self.logger,
	}

	if !entryCreated {
		ServerWithApi(self, "GET /",
			func(server *Server, request *Request, response *Response) {
				SendStatus(response, 404)
			},
		)
	}

	var waiter sync.WaitGroup

	waiter.Add(2)

	go func() {
		address := fmt.Sprintf("%s:%d", self.hostName, self.port)
		self.logger.Printf("listening for requests at http://%s", address)
		err := http.ListenAndServe(address, self.mux)
		if err != nil {
			if errors.Is(err, http.ErrServerClosed) {
				self.logger.Println("shutting down server")
				return
			}
			panic(err.Error())
		}
	}()

	go func() {
		secureAddress := fmt.Sprintf("%s:%d", self.hostName, self.securePort)
		if "" != self.certificate && "" != self.certificateKey {
			self.logger.Printf("listening for requests at https://%s", secureAddress)
			err := http.ListenAndServeTLS(secureAddress, self.certificate, self.certificateKey, self.mux)
			if err != nil {
				if errors.Is(err, http.ErrServerClosed) {
					self.logger.Println("shutting down server")
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
	pageId   string
	callback func(server *Server, request *Request, response *Response)
	mount    func(pattern string)
}

// routeCreate creates a route configuration from a callback function.
func routeCreate(
	callback func(server *Server, request *Request, response *Response),
) *Route {
	return &Route{
		isPage:   false,
		pageId:   "",
		callback: callback,
		mount:    func(pattern string) {},
	}
}

// pageRouteCreate creates a route configuration from a callback function, just like routeCreate.
//
// Unlike routeCreate, pageRouteCreate also creates a Page, which is used to automatically
// to serve a svelte page after invoking callback.
//
// Generally speaking, you should never manually invoke SendEcho or similar functions.
//
// However, it is safe to invoke receive functions, like ReceiveHeader, ReceiveCookie, etc.
func pageRouteCreate(
	pageId string,
	callback func(server *Server, request *Request, response *Response, page *Page),
) *Route {
	var pattern string
	if strings.HasSuffix(pageId, ".svelte") {
		pageId = strings.TrimSuffix(pageId, ".svelte")
	}

	return &Route{
		isPage: true,
		pageId: pageId,
		callback: func(server *Server, request *Request, response *Response) {
			page := &Page{
				renderMode:         RenderModeFull,
				data:               map[string]interface{}{},
				pageId:             pageId,
				embeddedFileSystem: server.embeddedFileSystem,
			}

			callback(server, request, response, page)

			if nil == page {
				ServerNotifyError(server, fmt.Errorf("svelte page handler `%s` returned a nil page", pattern))
				return
			}

			if nil == page.data {
				page.data = map[string]any{}
			}

			parseMultipartFormError := request.HttpRequest.ParseMultipartForm(1024)
			if parseMultipartFormError != nil {
				if !errors.Is(parseMultipartFormError, http.ErrNotMultipart) {
					ServerNotifyError(server, parseMultipartFormError)
				}

				parseFormError := request.HttpRequest.ParseForm()
				if parseFormError != nil {
					ServerNotifyError(server, parseFormError)
				}
			}

			pathEntry := map[string]string{}
			for _, name := range pathParametersPattern.FindAllStringSubmatch(pattern, -1) {
				if len(name) < 1 {
					continue
				}
				pathEntry[name[1]] = request.HttpRequest.PathValue(name[1])
			}
			page.data["path"] = pathEntry
			page.data["query"] = request.HttpRequest.URL.Query()
			page.data["form"] = request.HttpRequest.Form

			if VerifyAccept(request, "application/json") {
				data, marshalError := json.Marshal(page.data)
				if marshalError != nil {
					ServerNotifyError(server, marshalError)
					return
				}
				SendHeader(response, "Content-Type", "application/json")
				SendEcho(response, string(data))
				return
			}

			SendPage(response, page)
		},
		mount: func(patternLocal string) {
			pattern = patternLocal
			patternParts := strings.Split(patternLocal, " ")
			patternCounter := len(patternParts)
			if patternCounter > 1 {
				pagesToPaths[pageId] = path.Join(patternParts[1:]...)
			}
		},
	}
}

var entryCreated = false

// serverWithRoute registers a callback for the given pattern.
//
// If the given pattern conflicts with one that is already registered, serverWithRoute panics.
func serverWithRoute(
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
			HttpRequest: httpRequest,
		}

		httpHeader := writer.Header()

		response := Response{
			server:                self,
			writer:                &writer,
			lockedStatusAndHeader: false,
			statusCode:            200,
			header:                &httpHeader,
		}

		request.response = &response
		response.request = &request

		if isEntry {
			SendEmbeddedFileOrElse(&response, func() {
				SendFileOrElse(&response, func() {
					if route.callback != nil {
						route.callback(self, &request, &response)
					}
				})
			})
		} else if route.callback != nil {
			route.callback(self, &request, &response)
		}
	})
}

type Request struct {
	server      *Server
	response    *Response
	HttpRequest *http.Request
	webSocket   *websocket.Conn
}

type Response struct {
	server                *Server
	request               *Request
	writer                *http.ResponseWriter
	lockedStatusAndHeader bool
	statusCode            int
	header                *http.Header
	webSocket             *websocket.Conn
}

// ServerWithErrorReceiver sets the error receiver.
func ServerWithErrorReceiver(self *Server, callback func(err error)) {
	self.errorHandler = callback
}

// ServerNotifyError notifies the server of an error.
func ServerNotifyError(self *Server, err error) {
	self.errorHandler(err)
}

// SendRedirect redirects the request.
func SendRedirect(self *Response, location string, statusCode int) {
	SendStatus(self, statusCode)
	SendHeader(self, "Location", location)
	SendEcho(self, "")
}

// SendRedirectToSecure tries to redirect the request to the https server.
//
// When the request is already secure, SendRedirectToSecure returns false.
func SendRedirectToSecure(self *Response, statusCode int) bool {
	request := self.request
	if "" == request.server.certificate || "" == request.server.certificateKey || request.HttpRequest.TLS != nil {
		return false
	}

	insecureSuffix := fmt.Sprintf(":%d", request.server.port)
	secureSuffix := fmt.Sprintf(":%d", request.server.securePort)
	secureHost := strings.Replace(request.HttpRequest.Host, insecureSuffix, secureSuffix, 1)
	secureLocation := fmt.Sprintf("https://%s%s", secureHost, request.HttpRequest.RequestURI)
	SendRedirect(self, secureLocation, statusCode)
	return true
}

// SendStatus sets the status code.
//
// This will lock the status, which makes it
// so that the next time you invoke this
// function it will fail with an error.
//
// You can retrieve the error using ServerWithErrorReceiver.
func SendStatus(self *Response, code int) {
	if self.lockedStatusAndHeader {
		ServerNotifyError(self.server, errors.New("status is locked"))
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
// You can retrieve the error using ServerWithErrorReceiver
func SendHeader(self *Response, key string, value string) {
	if self.lockedStatusAndHeader {
		ServerNotifyError(self.server, errors.New("headers locked"))
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
	SendHeader(self, "set-Cookie", fmt.Sprintf("%s=%s", url.QueryEscape(key), url.QueryEscape(value)))
}

// SendContent sends binary safe content.
//
// If the status code or the header have not been sent already, a default status of "200 OK" will be sent immediately along with whatever headers you've previously defined.
//
// The status code and the header will become locked and further attempts to send either of them will fail with an error.
//
// You can retrieve the error using ServerWithErrorReceiver.
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
			ServerNotifyError(self.server, writeError)
		}
		return
	}

	_, err := (*self.writer).Write(content)
	if err != nil {
		ServerNotifyError(self.server, err)
		return
	}
}

// SendEcho sends utf-8 safe content.
//
// If the status code or the header have not been sent already, a default status of "200 OK" will be sent immediately along with whatever headers you've previously defined.
//
// The status code and the header will become locked and further attempts to send either of them will fail with an error.
//
// You can retrieve the error using ServerWithErrorReceiver.
//
// Compatible with web sockets.
func SendEcho(self *Response, content string) {
	SendContent(self, []byte(content))
}

// SendJson sends json content.
//
// If the status code or the header have not been sent already, a default status of "200 OK" will be sent immediately along with whatever headers you've previously defined.
//
// The status code and the header will become locked and further attempts to send either of them will fail with an error.
//
// You can retrieve the error using ServerWithErrorReceiver.
//
// Compatible with web sockets.
func SendJson(self *Response, payload any) {
	content, marshalError := json.Marshal(payload)
	if marshalError != nil {
		ServerNotifyError(self.server, marshalError)
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
	requestedMime := self.HttpRequest.Header.Get("Content-Type")
	for _, acceptedMime := range contentTypes {
		if acceptedMime == "*" || strings.HasPrefix(requestedMime, acceptedMime) {
			return true
		}
	}

	return false
}

// VerifyAccept checks if the incoming request accepts any of the given content-types.
func VerifyAccept(self *Request, contentTypes ...string) bool {
	requestedAcceptMime := self.HttpRequest.Header.Get("Accept")
	for _, acceptedMime := range contentTypes {
		if acceptedMime == "*" || strings.Contains(requestedAcceptMime, acceptedMime) {
			return true
		}
	}

	return false
}

// SendEmbeddedFileOrIndexOrElse sends the embedded file requested by the client,
// or the closest index.html embedded file, or else falls back.
func SendEmbeddedFileOrIndexOrElse(self *Response, orElse func()) {
	request := self.request
	fileName := filepath.Join(".dist", "client", request.HttpRequest.RequestURI)

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
		ServerNotifyError(self.server, readerError)
		return
	}

	SendHeader(self, "Content-Type", Mime(fileName))
	SendHeader(self, "Content-Length", fmt.Sprintf("%d", (*info).Size()))
	http.ServeContent(*self.writer, request.HttpRequest, fileName, (*info).ModTime(), reader)
}

// SendEmbeddedFileOrElse sends the embedded file requested by the client,
// or the closest index.html embedded file, or else falls back.
func SendEmbeddedFileOrElse(self *Response, orElse func()) {
	request := self.request
	fileName := filepath.Join(".dist", "client", request.HttpRequest.RequestURI)
	fileName = strings.Split(fileName, "?")[0]
	fileName = strings.Split(fileName, "&")[0]

	if !EmbeddedExists(request.server.embeddedFileSystem, fileName) ||
		EmbeddedIsDirectory(request.server.embeddedFileSystem, fileName) {
		orElse()
		return
	}

	reader, info, readerError := createReaderFromEmbeddedFileName(&request.server.embeddedFileSystem, fileName)
	if readerError != nil {
		ServerNotifyError(self.server, readerError)
		return
	}

	SendHeader(self, "Content-Type", Mime(fileName))
	SendHeader(self, "Content-Length", fmt.Sprintf("%d", (*info).Size()))
	http.ServeContent(*self.writer, request.HttpRequest, fileName, (*info).ModTime(), reader)
}

// SendFileOrIndexOrElse sends the file requested by the client,
// or the closest index.html file, or else falls back.
func SendFileOrIndexOrElse(self *Response, orElse func()) {
	request := self.request
	fileName := filepath.Join(".dist", "client", request.HttpRequest.RequestURI)

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
		ServerNotifyError(self.server, readerError)
		return
	}

	SendHeader(self, "Content-Type", Mime(fileName))
	SendHeader(self, "Content-Length", fmt.Sprintf("%d", (*info).Size()))
	http.ServeContent(*self.writer, request.HttpRequest, fileName, (*info).ModTime(), reader)
}

// SendFileOrElse sends the file requested by the client, or else falls back.
func SendFileOrElse(self *Response, orElse func()) {
	request := self.request
	fileName := filepath.Join(".dist", "client", request.HttpRequest.RequestURI)

	if !Exists(fileName) || IsDirectory(fileName) {
		orElse()
		return
	}

	reader, info, readerError := createReaderFromFileName(fileName)
	if readerError != nil {
		ServerNotifyError(self.server, readerError)
		return
	}

	SendHeader(self, "Content-Type", Mime(fileName))
	SendHeader(self, "Content-Length", fmt.Sprintf("%d", (*info).Size()))
	http.ServeContent(*self.writer, request.HttpRequest, fileName, (*info).ModTime(), reader)
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
	conn, upgradeError := self.server.webSocketUpgrader.Upgrade(*self.writer, request.HttpRequest, nil)
	if upgradeError != nil {
		ServerNotifyError(request.server, upgradeError)
		return
	}
	defer conn.Close()
	self.webSocket = conn
	request.webSocket = conn
	self.lockedStatusAndHeader = true
	callback()
}

func serverSqlFindNextFallback(dest ...any) bool { return false }
func severSqlFindCloseFallback()                 {}

// ServerSqlExecute executes sql queries that don't return rows, typically INSERT, UPDATE, DELETE queries.
func ServerSqlExecute(self *Server, query string, props ...any) *sql.Result {
	transaction, transactionError := self.database.Begin()
	if transactionError != nil {
		ServerNotifyError(self, transactionError)
		return nil
	}

	result, execError := transaction.Exec(query, props...)
	if execError != nil {
		ServerNotifyError(self, execError)
		rollbackError := transaction.Rollback()
		if rollbackError != nil {
			ServerNotifyError(self, rollbackError)
		}
		return nil
	}

	commitError := transaction.Commit()
	if commitError != nil {
		ServerNotifyError(self, commitError)
		return nil
	}

	return &result
}

// ServerSqlFind executes a sql query that returns rows, typically a SELECT query.
//
// It returns a next function and a close function.
//
// Use next to project the next row onto dest.
//
// Next will return false if where are no more rows available.
//
// Use close to close the database context and prevent any subsequent enumerations.
//
// Whenever next returns false, the database context is closed automatically as if calling close.
func ServerSqlFind(self *Server, query string, props ...any) (next func(dest ...any) bool, close func()) {
	next = serverSqlFindNextFallback
	close = severSqlFindCloseFallback

	rows, queryError := self.database.Query(query, props...)
	if queryError != nil {
		ServerNotifyError(self, queryError)
		return
	}

	next = func(dest ...any) bool {
		if !rows.Next() {
			return false
		}

		err := rows.Scan(dest...)
		if err != nil {
			return false
		}
		return true
	}
	close = func() {
		err := rows.Close()
		if err != nil {
			ServerNotifyError(self, err)
		}
	}
	return
}

// ServerSqlCreateTable creates a table from a type.
func ServerSqlCreateTable[Table any](self *Server) {
	var query strings.Builder
	t := reflect.TypeFor[Table]()
	query.WriteString(fmt.Sprintf("create table `%s`(\n", t.Name()))
	count := t.NumField()
	for i := 0; i < count; i++ {
		field := t.Field(i)
		rules := field.Tag.Get("sql")
		if i > 0 {
			query.WriteString(",\n")
		}
		query.WriteString(fmt.Sprintf("`%s` %s", field.Name, rules))
	}
	query.WriteString(");")
	_, err := self.database.Exec(query.String())
	if err != nil {
		ServerNotifyError(self, err)
	}
}

type svelteRouterProps struct {
	PageId string            `json:"pageId"`
	Data   map[string]any    `json:"data"`
	Paths  map[string]string `json:"paths"`
}

type Page struct {
	renderMode         RenderMode
	data               map[string]any
	headless           bool
	embeddedFileSystem embed.FS
	pageId             string
}

func PageWithRenderMode(self *Page, renderMode RenderMode) {
	self.renderMode = renderMode
}

func PageWithData(self *Page, key string, value any) {
	self.data[key] = value
}

var noScriptPattern = regexp.MustCompile(`<script.*>.*</script>`)
var pagesToPaths = map[string]string{}

// PageCompile compiles a svelte page.
func PageCompile(self *Page) (string, error) {
	if nil == self {
		self = &Page{
			renderMode: RenderModeFull,
			data:       map[string]any{},
			headless:   false,
		}
	} else {
		if nil == self.data {
			self.data = map[string]any{}
		}
	}

	fileNameIndex := filepath.Join(".dist", "client", ".frizzante", "vite-project", "index.html")

	var indexBytes []byte

	if "1" == os.Getenv("DEV") {
		indexBytesLocal, readError := os.ReadFile(fileNameIndex)
		if readError != nil {
			return "", readError
		}
		indexBytes = indexBytesLocal
	} else {
		indexBytesLocal, readError := self.embeddedFileSystem.ReadFile(fileNameIndex)
		if readError != nil {
			return "", readError
		}
		indexBytes = indexBytesLocal
	}

	routerPropsBytes, jsonError := json.Marshal(svelteRouterProps{
		PageId: self.pageId,
		Data:   self.data,
		Paths:  pagesToPaths,
	})

	if jsonError != nil {
		return "", jsonError
	}

	routerPropsString := string(routerPropsBytes)

	targetId, targetIdError := uuid.NewV4()
	if targetIdError != nil {
		return "", targetIdError
	}

	if self.headless {
		_, body, renderError := render(self.embeddedFileSystem, routerPropsString)
		if renderError != nil {
			return "", renderError
		}
		return body, nil
	}

	var index string
	if RenderModeFull == self.renderMode {
		head, body, renderError := render(self.embeddedFileSystem, routerPropsString)
		if renderError != nil {
			return "", renderError
		}
		index = strings.Replace(
			strings.Replace(
				strings.Replace(
					strings.Replace(
						string(indexBytes),
						"<!--app-target-->",
						fmt.Sprintf(`<script type="application/javascript">function target(){return document.getElementById("%s")}</script>`, targetId),
						1,
					),
					"<!--app-body-->",
					fmt.Sprintf(`<div id="%s">%s</div>`, targetId, body),
					1,
				),
				"<!--app-head-->",
				head,
				1,
			),
			"<!--app-data-->",
			fmt.Sprintf(
				`<script type="application/javascript">function props(){return %s}</script>`,
				routerPropsString,
			),
			1,
		)
	} else if RenderModeClient == self.renderMode {
		index = strings.Replace(
			strings.Replace(
				strings.Replace(
					strings.Replace(
						string(indexBytes),
						"<!--app-target-->",
						fmt.Sprintf(`<script type="application/javascript">function target(){return document.getElementById("%s")}</script>`, targetId),
						1,
					),
					"<!--app-body-->",
					fmt.Sprintf(`<div id="%s"></div>`, targetId),
					1,
				),
				"<!--app-head-->",
				"",
				1,
			),
			"<!--app-data-->",
			fmt.Sprintf(
				`<script type="application/javascript">function props(){return %s}</script>`,
				routerPropsString,
			),
			1,
		)
	} else if RenderModeServer == self.renderMode {
		head, body, renderError := render(self.embeddedFileSystem, routerPropsString)
		if renderError != nil {
			return "", renderError
		}
		index = strings.Replace(
			strings.Replace(
				strings.Replace(
					strings.Replace(
						noScriptPattern.ReplaceAllString(string(indexBytes), ""),
						"<!--app-target-->",
						``,
						1,
					),
					"<!--app-body-->",
					fmt.Sprintf(`<div id="%s">%s</div>`, targetId, body),
					1,
				),
				"<!--app-head-->",
				head,
				1,
			),
			"<!--app-data-->",
			"",
			1,
		)
	}

	return index, nil
}

// PageCreate creates a page.
func PageCreate(
	renderMode RenderMode,
	data map[string]any,
) *Page {
	return &Page{
		renderMode: renderMode,
		data:       data,
		headless:   false,
	}
}

// PageHeadlessCreate creates a headless page.
func PageHeadlessCreate(data map[string]any) *Page {
	return &Page{
		renderMode: RenderModeServer,
		data:       data,
		headless:   true,
	}
}

// SendPage renders and echos a svelte page.
func SendPage(self *Response, page *Page) {
	content, compileError := PageCompile(page)
	if nil != compileError {
		ServerNotifyError(self.server, compileError)
		return
	}

	contentType := ReceiveContentType(self.request)

	if page.headless {
		if "" == contentType {
			contentType = "text/html"
		}

		content = strings.Replace(content, "<!--[-->", "", -1)
		content = strings.Replace(content, "<!--]-->", "", -1)
		content = strings.Replace(content, "<!---->", "", -1)
	}

	SendHeader(self, "Content-Type", contentType)
	SendEcho(self, content)
}

// ServerWithSessionOperator sets the session operator,
// which is a function that provides the four main
// operations used by the server to manage any session,
// get, set, unset and destroy.
//
// Get must retrieve data from the session store.
//
// Set must create a new property to the session store or update an existing one.
//
// Unset must remove a property from the session store.
//
// Destroy must destroy the whole session, store included.
//
// In this context, "store", is any type of data storage,
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

// ServerWithApi maps an api route.
func ServerWithApi(self *Server,
	pattern string,
	callback func(
	server *Server,
	request *Request,
	response *Response,
),
) {
	serverWithRoute(self, pattern, routeCreate(callback))
}

// ServerWithPage maps a page route.
func ServerWithPage(self *Server,
	pattern string,
	pageId string,
	callback func(
	server *Server,
	request *Request,
	response *Response,
	page *Page,
),
) {
	serverWithRoute(self, pattern, pageRouteCreate(pageId, callback))
}

// ServerWithHeadlessPage maps a page route and renders it in headless mode.
//
// Rendering a headless page means it's automatically rendering in server mode,
// it omits the head tag, the body tag and all css.
func ServerWithHeadlessPage(
	self *Server,
	pattern string,
	pageId string,
	callback func(
	server *Server,
	request *Request,
	response *Response,
	page *Page,
),
) {
	serverWithRoute(self, pattern,
		pageRouteCreate(pageId,
			func(server *Server, request *Request, response *Response, page *Page) {
				page.headless = true
				callback(server, request, response, page)
			},
		),
	)
}
