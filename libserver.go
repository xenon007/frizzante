package frizzante

import (
	"crypto/tls"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Server struct {
	hostname           string
	port               int
	server             *http.Server
	mux                *http.ServeMux
	sessions           map[string]*net.Conn
	readTimeout        time.Duration
	writeTimeout       time.Duration
	maxHeaderBytes     int
	errorLogger        *log.Logger
	informationLogger  *log.Logger
	tlsConfiguration   *tls.Config
	informationHandler []func(string)
	errorHandler       []func(error)
	temporaryDirectory string
	pagesDirectory     string
	pageExtension      string
}

// ServerCreate creates a server.
func ServerCreate() *Server {
	return &Server{
		hostname:           "",
		port:               80,
		server:             nil,
		mux:                http.NewServeMux(),
		sessions:           map[string]*net.Conn{},
		readTimeout:        10 * time.Second,
		writeTimeout:       10 * time.Second,
		maxHeaderBytes:     3 * mb,
		errorLogger:        log.Default(),
		informationLogger:  log.Default(),
		tlsConfiguration:   nil,
		informationHandler: []func(string){},
		errorHandler:       []func(error){},
		temporaryDirectory: ".temp",
		pagesDirectory:     "pages",
		pageExtension:      "svelte",
	}
}

// ServerWithHostname sets the server hostname.
func ServerWithHostname(self *Server, hostname string) {
	self.hostname = hostname
}

// ServerWithPort sets the server port.
func ServerWithPort(self *Server, port int) {
	self.port = port
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

// ServerWithTlsConfiguration sets the tls configuration.
func ServerWithTlsConfiguration(self *Server, tlsConfiguration *tls.Config) {
	self.tlsConfiguration = tlsConfiguration
}

// ServerWithTemporaryDirectory sets the temporary directory.
func ServerWithTemporaryDirectory(self *Server, temporaryDirectory string) {
	self.temporaryDirectory = temporaryDirectory
}

// ServerWithPagesDirectory sets the pages directory.
//
// Default is "pages".
func ServerWithPagesDirectory(self *Server, pagesDirectory string) {
	self.pagesDirectory = pagesDirectory
}

// ServerWithPageExtension sets an extension name for server pages.
//
// Default is "svelte".
func ServerWithPageExtension(self *Server, pageExtension string) {
	self.pageExtension = pageExtension
}

// ServerGetPage gets a server page.
//
// Extension name must be omitted.
//
// You can use ServerWithPageExtension to modify pages extension.
//
// Any errors will be forwarded to the server logger silently.
//
// Output will be blank if an error occurs.
//
// When id is longer than 255 characters, the operation will fail silently and the server will be notified.
func ServerGetPage(self *Server, id string) string {
	if len(id) > 255 {
		ServerNotifyError(self, fmt.Errorf("page id is too long"))
		return ""
	}

	if strings.Contains(id, "./") {
		ServerNotifyError(self, fmt.Errorf("invalid substring `./` detected in page id `%s`", id))
		return ""
	}

	if strings.Contains(id, "../") {
		ServerNotifyError(self, fmt.Errorf("invalid substring `../` detected in page id `%s`", id))
		return ""
	}

	fileName := self.pagesDirectory
	if !strings.HasSuffix(fileName, "/") && !strings.HasPrefix(id, "/") {
		fileName += "/"
	}

	if !strings.HasSuffix(self.pageExtension, ".") {
		fileName += id + "." + self.pageExtension
	} else {
		fileName += id + self.pageExtension
	}

	contents, err := os.ReadFile(fileName)
	if err != nil {
		ServerNotifyError(self, err)
		return ""
	}
	return string(contents)
}

// ServerHasPage checks if a page file exists.
//
// When id is longer than 255 characters, the operation will fail silently and the server will be notified.
func ServerHasPage(self *Server, id string) bool {
	if len(id) > 255 {
		ServerNotifyError(self, fmt.Errorf("page id is too long"))
		return false
	}

	if strings.Contains(id, "./") || strings.Contains(id, "../") {
		return false
	}

	fileName := self.pagesDirectory
	if !strings.HasSuffix(fileName, "/") && !strings.HasPrefix(id, "/") {
		fileName += "/"
	}
	fileName += id
	return exists(fileName)
}

// ServerSetTemporaryFile sets a temporary file.
//
// When id is longer than 255 characters, the operation will fail silently and the server will be notified.
func ServerSetTemporaryFile(self *Server, id string, contents string) {
	if len(id) > 255 {
		ServerNotifyError(self, fmt.Errorf("temporary file id is too long"))
		return
	}

	if strings.Contains(id, "./") {
		ServerNotifyError(self, fmt.Errorf("invalid substring `./` detected in temporary file id `%s`", id))
		return
	}

	if strings.Contains(id, "../") {
		ServerNotifyError(self, fmt.Errorf("invalid substring `../` detected in temporary file id `%s`", id))
		return
	}

	if !exists(self.temporaryDirectory) {
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
	if !exists(directory) {
		mkdirError := os.MkdirAll(directory, os.ModePerm)
		if mkdirError != nil {
			ServerNotifyError(self, mkdirError)
			return
		}
	}

	var file *os.File

	if !ServerHasTemporaryFile(self, id) {
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

// ServerGetTemporaryFile gets a temporary file.
func ServerGetTemporaryFile(self *Server, id string) string {
	if strings.Contains(id, "./") {
		ServerNotifyError(self, fmt.Errorf("invalid substring `./` detected in temporary file id `%s`", id))
		return ""
	}

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

// ServerHasTemporaryFile checks if a temporary file exists.
//
// When id is longer than 255 characters, the operation will fail silently and the server will be notified.
func ServerHasTemporaryFile(self *Server, id string) bool {
	if len(id) > 255 {
		ServerNotifyError(self, fmt.Errorf("temporary file id is too long"))
		return false
	}

	if strings.Contains(id, "./") || strings.Contains(id, "../") {
		return false
	}

	fileName := self.temporaryDirectory
	if !strings.HasSuffix(fileName, "/") && !strings.HasPrefix(id, "/") {
		fileName += "/"
	}
	fileName += id
	return exists(fileName)
}

// ServerClearTemporaryDirectory clears the temporary directory.
func ServerClearTemporaryDirectory(self *Server) {
	err := os.RemoveAll(self.temporaryDirectory)
	if err != nil {
		ServerNotifyError(self, err)
	}
}

// ServerStart starts the server.
//
// If for some reason the server cannot start, ServerStarts panics.
func ServerStart(self *Server) {
	self.server = &http.Server{
		Handler:        self.mux,
		ReadTimeout:    self.readTimeout,
		WriteTimeout:   self.writeTimeout,
		MaxHeaderBytes: self.maxHeaderBytes,
		ErrorLog:       self.errorLogger,
		TLSConfig:      self.tlsConfiguration,
	}

	address := fmt.Sprintf("%s:%d", self.hostname, self.port)

	self.informationLogger.Printf("Listening for requests at https://%s", address)

	err := http.ListenAndServe(address, self.mux)
	if err != nil {
		panic(err.Error())
	}
}

// ServerOnRequest registers a handler function for the given pattern.
//
// If the given pattern conflicts, with one that is already registered, ServerOnRequest panics.
func ServerOnRequest(
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
}

type Response struct {
	server                *Server
	writer                *http.ResponseWriter
	lockedStatusAndHeader bool
	statusCode            int
	header                *http.Header
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

// Status sets the status code.
//
// The status message will be inferred automatically based on the code.
//
// This will lock the status, which makes it
// so that the next time you invoke this
// function it will fail with an error.
//
// You can retrieve this error using ServerOnError.
func Status(self *Response, code int) {
	if self.lockedStatusAndHeader {
		ServerNotifyError(self.server, errors.New("status is locked"))
		return
	}
	self.statusCode = code
}

// Header sets a header field.
//
// If the status has not been sent already, a default "200 OK" status will be sent immediately.
//
// This means the status will become locked and further attempts to send the status will fail with an error.
//
// You can retrieve this error using ServerOnError
func Header(self *Response, key string, value string) {
	if self.lockedStatusAndHeader {
		ServerNotifyError(self.server, errors.New("headers locked"))
		return
	}

	self.header.Set(key, value)
}

// Send sends binary safe content.
//
// If the status code or the header have not been sent already, a default status of "200 OK" will be sent immediately along with whatever headers you've previously defined.
//
// The status code and the header will become locked and further attempts to send either of them will fail with an error.
//
// You can retrieve this error using ServerOnError.
func Send(self *Response, value []byte) {
	writer := *self.writer

	if !self.lockedStatusAndHeader {
		writer.WriteHeader(self.statusCode)
		self.lockedStatusAndHeader = true
	}

	_, err := writer.Write(value)
	if err != nil {
		ServerNotifyError(self.server, err)
		return
	}
}

// Echo sends utf-8 safe content.
//
// If the status code or the header have not been sent already, a default status of "200 OK" will be sent immediately along with whatever headers you've previously defined.
//
// The status code and the header will become locked and further attempts to send either of them will fail with an error.
//
// You can retrieve this error using ServerOnError.
//
// See fmt.Sprintf.
func Echo(self *Response, content string) {
	Send(self, []byte(content))
}

// Accept checks if the incoming request specifies specific content-types.
func Accept(self *Request, mimes ...string) bool {
	requestedMime := self.HttpRequest.Header.Get("content-type")
	for _, acceptedMime := range mimes {
		if acceptedMime == "*" || strings.HasPrefix(requestedMime, acceptedMime) {
			return true
		}
	}

	return false
}
