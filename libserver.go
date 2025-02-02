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
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"rogchap.com/v8go"
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
	errorLogger            *log.Logger
	informationLogger      *log.Logger
	certificate            string
	certificateKey         string
	informationHandler     []func(string)
	errorHandler           []func(error)
	temporaryDirectory     string
	embeddedFileSystem     embed.FS
	webSocketUpgrader      *websocket.Upgrader
}

// ServerCreate creates a server.
func ServerCreate() *Server {
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
		errorLogger:        log.Default(),
		informationLogger:  log.Default(),
		certificate:        "",
		certificateKey:     "",
		informationHandler: []func(string){},
		errorHandler:       []func(error){},
		temporaryDirectory: ".temp",
		webSocketUpgrader: &websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
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

// ServerWithErrorLogger sets the error logger.
func ServerWithErrorLogger(self *Server, errorLogger *log.Logger) {
	self.errorLogger = errorLogger
}

// ServerWithInformationLogger sets the information logger.
func ServerWithInformationLogger(self *Server, informationLogger *log.Logger) {
	self.informationLogger = informationLogger
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
func ReceiveQuery(self *Request, name string) string {
	return self.HttpRequest.URL.Query().Get(name)
}

// ReceivePath reads a path fields and returns the value.
func ReceivePath(self *Request, name string) string {
	return self.HttpRequest.PathValue(name)
}

// ReceiveHeader reads a header field and returns the value.
func ReceiveHeader(self *Request, key string) string {
	return self.HttpRequest.Header.Get(key)
}

// ReceiveContentType reads the Content-Type header field and returns the value.
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
		ErrorLog:       self.errorLogger,
	}

	var waiter sync.WaitGroup

	waiter.Add(2)

	go func() {
		address := fmt.Sprintf("%s:%d", self.hostName, self.port)
		self.informationLogger.Printf("listening for requests at http://%s", address)
		err := http.ListenAndServe(address, self.mux)
		if err != nil {
			if errors.Is(err, http.ErrServerClosed) {
				self.informationLogger.Println("shutting down server")
				return
			}
			panic(err.Error())
		}
	}()

	go func() {
		secureAddress := fmt.Sprintf("%s:%d", self.hostName, self.securePort)
		if "" != self.certificate && "" != self.certificateKey {
			self.informationLogger.Printf("listening for requests at https://%s", secureAddress)
			err := http.ListenAndServeTLS(secureAddress, self.certificate, self.certificateKey, self.mux)
			if err != nil {
				if errors.Is(err, http.ErrServerClosed) {
					self.informationLogger.Println("shutting down server")
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

// ServerWithSveltePage creates a request handler that serves a svelte page.
func ServerWithSveltePage(self *Server, pattern string, pageId string, configure func(*Server, *Request, *Response) *SveltePageConfiguration) {
	if strings.HasSuffix(pageId, ".svelte") {
		pageId = strings.TrimSuffix(pageId, ".svelte")
	}
	patternParts := strings.Split(pattern, " ")
	if len(patternParts) > 1 {
		sveltePagesToPaths[pageId] = patternParts[1]
	}

	ServerWithRequestHandler(self, pattern, func(server *Server, request *Request, response *Response) {
		SendEmbeddedFileOrElse(response, request, func() {
			SendFileOrElse(response, request, func() {
				configuration := configure(server, request, response)

				if nil == configuration.Globals {
					configuration.Globals = map[string]v8go.FunctionCallback{}
				}

				if nil == configuration.Data {
					configuration.Data = map[string]any{}
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

				path := map[string]string{}
				for _, name := range pathParametersPattern.FindAllStringSubmatch(pattern, -1) {
					if len(name) < 1 {
						continue
					}
					path[name[1]] = request.HttpRequest.PathValue(name[1])
				}
				configuration.Data["path"] = path
				configuration.Data["query"] = request.HttpRequest.URL.Query()
				configuration.Data["form"] = request.HttpRequest.Form
				SendSveltePage(response, pageId, configuration)
			})
		})
	})
}

// ServerWithWebSocketHandler upgrades all incoming requests to the given pattern
// to web sockets and invokes the callback before closing the connection.
//
// If the given pattern conflicts with one that is already registered,
// or it uses a verb other than GET, ServerWithWebSocketHandler panics.
func ServerWithWebSocketHandler(self *Server, pattern string, callback func(request *Request, response *Response)) {
	if !strings.HasPrefix(pattern, "GET ") {
		panic(fmt.Errorf("all web socket patterns must be defined using the GET http verb, received `%s` instead", pattern))
		return
	}
	ServerWithRequestHandler(self, pattern, func(server *Server, request *Request, response *Response) {
		SendWebSocketUpgrade(response, request, callback)
	})
}

// ServerWithRequestHandler registers a callback for the given pattern.
//
// If the given pattern conflicts with one that is already registered, ServerWithRequestHandler panics.
func ServerWithRequestHandler(
	self *Server,
	pattern string,
	callback func(server *Server, request *Request, response *Response),
) {
	self.mux.HandleFunc(pattern, func(writer http.ResponseWriter, request *http.Request) {
		requestLocal := Request{
			server:      self,
			HttpRequest: request,
		}

		httpHeader := writer.Header()

		responseLocal := Response{
			server:                self,
			writer:                &writer,
			lockedStatusAndHeader: false,
			statusCode:            200,
			header:                &httpHeader,
		}
		callback(self, &requestLocal, &responseLocal)
	})
}

type Request struct {
	server      *Server
	HttpRequest *http.Request
	webSocket   *websocket.Conn
}

type Response struct {
	server                *Server
	writer                *http.ResponseWriter
	lockedStatusAndHeader bool
	statusCode            int
	header                *http.Header
	webSocket             *websocket.Conn
}

// ServerOnInformation handles server information.
func ServerOnInformation(self *Server, callback func(information string)) {
	self.informationHandler = append(self.informationHandler, callback)
}

// ServerOnError handles server errors.
func ServerOnError(self *Server, callback func(err error)) {
	self.errorHandler = append(self.errorHandler, callback)
}

// ServerNotifyInformation notifies the server of some information.
func ServerNotifyInformation(self *Server, information string) {
	for _, listener := range self.informationHandler {
		listener(information)
	}
}

// ServerNotifyError notifies the server of an error.
func ServerNotifyError(self *Server, err error) {
	for _, listener := range self.errorHandler {
		listener(err)
	}
}

// ServerLogInformation logs information using the server's logger.
func ServerLogInformation(self *Server, information string) {
	self.informationLogger.Println(information)
}

// ServerLogError logs an error using the server's logger.
func ServerLogError(self *Server, err error) {
	self.errorLogger.Println(err.Error())
}

// SendRedirect redirects the request.
func SendRedirect(response *Response, location string, statusCode int) {
	SendStatus(response, statusCode)
	SendHeader(response, "Location", location)
	SendEcho(response, "")
}

// SendRedirectToSecure tries to redirect the request to the https server.
//
// When the request is already secure, SendRedirectToSecure returns false.
func SendRedirectToSecure(request *Request, response *Response, statusCode int) bool {
	if "" == request.server.certificate || "" == request.server.certificateKey || request.HttpRequest.TLS != nil {
		return false
	}

	insecureSuffix := fmt.Sprintf(":%d", request.server.port)
	secureSuffix := fmt.Sprintf(":%d", request.server.securePort)
	secureHost := strings.Replace(request.HttpRequest.Host, insecureSuffix, secureSuffix, 1)
	secureLocation := fmt.Sprintf("https://%s%s", secureHost, request.HttpRequest.RequestURI)
	SendRedirect(response, secureLocation, statusCode)
	return true
}

// SendStatus sets the status code.
//
// This will lock the status, which makes it
// so that the next time you invoke this
// function it will fail with an error.
//
// You can retrieve this error using ServerOnError.
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
// You can retrieve this error using ServerOnError
func SendHeader(self *Response, key string, value string) {
	if self.lockedStatusAndHeader {
		ServerNotifyError(self.server, errors.New("headers locked"))
		return
	}

	self.header.Set(key, value)
}

// SendCookie sends a cookies to the client.
func SendCookie(self *Response, key string, value string) {
	SendHeader(self, "Set-Cookie", fmt.Sprintf("%s=%s", url.QueryEscape(key), url.QueryEscape(value)))
}

// SendContent sends binary safe content.
//
// If the status code or the header have not been sent already, a default status of "200 OK" will be sent immediately along with whatever headers you've previously defined.
//
// The status code and the header will become locked and further attempts to send either of them will fail with an error.
//
// You can retrieve this error using ServerOnError.
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
// You can retrieve this error using ServerOnError.
//
// See fmt.Sprintf.
func SendEcho(self *Response, content string) {
	SendContent(self, []byte(content))
}

// VerifyContentType checks if the incoming request has any of the given content-types.
func VerifyContentType(self *Request, contentTypes ...string) bool {
	requestedMime := self.HttpRequest.Header.Get("content-type")
	for _, acceptedMime := range contentTypes {
		if acceptedMime == "*" || strings.HasPrefix(requestedMime, acceptedMime) {
			return true
		}
	}

	return false
}

// SendEmbeddedFileOrIndexOrElse sends the embedded file requested by the client,
// or the closest index.html embedded file, or else falls back.
func SendEmbeddedFileOrIndexOrElse(response *Response, request *Request, orElse func()) {
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
		ServerNotifyError(response.server, readerError)
		return
	}

	SendHeader(response, "Content-Type", Mime(fileName))
	SendHeader(response, "Content-Length", fmt.Sprintf("%d", (*info).Size()))
	http.ServeContent(*response.writer, request.HttpRequest, fileName, (*info).ModTime(), reader)
}

// SendEmbeddedFileOrElse sends the embedded file requested by the client,
// or the closest index.html embedded file, or else falls back.
func SendEmbeddedFileOrElse(response *Response, request *Request, orElse func()) {
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
		ServerNotifyError(response.server, readerError)
		return
	}

	SendHeader(response, "Content-Type", Mime(fileName))
	SendHeader(response, "Content-Length", fmt.Sprintf("%d", (*info).Size()))
	http.ServeContent(*response.writer, request.HttpRequest, fileName, (*info).ModTime(), reader)
}

// SendFileOrIndexOrElse sends the file requested by the client,
// or the closest index.html file, or else falls back.
func SendFileOrIndexOrElse(response *Response, request *Request, orElse func()) {
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
		ServerNotifyError(response.server, readerError)
		return
	}

	SendHeader(response, "Content-Type", Mime(fileName))
	SendHeader(response, "Content-Length", fmt.Sprintf("%d", (*info).Size()))
	http.ServeContent(*response.writer, request.HttpRequest, fileName, (*info).ModTime(), reader)
}

// SendFileOrElse sends the file requested by the client, or else falls back.
func SendFileOrElse(response *Response, request *Request, orElse func()) {
	fileName := filepath.Join(".dist", "client", request.HttpRequest.RequestURI)

	if !Exists(fileName) || IsDirectory(fileName) {
		orElse()
		return
	}

	reader, info, readerError := createReaderFromFileName(fileName)
	if readerError != nil {
		ServerNotifyError(response.server, readerError)
		return
	}

	SendHeader(response, "Content-Type", Mime(fileName))
	SendHeader(response, "Content-Length", fmt.Sprintf("%d", (*info).Size()))
	http.ServeContent(*response.writer, request.HttpRequest, fileName, (*info).ModTime(), reader)
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
func SendWebSocketUpgrade(self *Response, request *Request, callback func(request *Request, response *Response)) {
	conn, upgradeError := self.server.webSocketUpgrader.Upgrade(*self.writer, request.HttpRequest, nil)
	if upgradeError != nil {
		ServerNotifyError(request.server, upgradeError)
		return
	}
	defer conn.Close()
	self.webSocket = conn
	request.webSocket = conn
	self.lockedStatusAndHeader = true
	callback(request, self)
}

// ServerSqlExecute executes a sql query and returns a function that can be used to extract the results.
//
// The resulting function advances the row and copies the columns into the values pointed at by dest.
//
// ServerSqlExecute also returns a second function, which when invoked will close
// the internal query result preventing further enumerations.
func ServerSqlExecute(self *Server, query string, props ...any) (func(dest ...any) bool, func()) {
	rows, execError := self.database.Query(query, props...)
	if execError != nil {
		ServerNotifyError(self, execError)
		return func(columns ...any) bool { return false }, func() {}
	}

	return func(columns ...any) bool {
			if !rows.Next() {
				return false
			}
			err := rows.Scan(columns...)
			if err != nil {
				return false
			}
			return true
		},
		func() {
			err := rows.Close()
			if err != nil {
				ServerNotifyError(self, err)
			}
		}
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
