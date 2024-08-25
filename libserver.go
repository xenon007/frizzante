package frizzante

import (
	"crypto/tls"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
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
}

// Create a server.
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
	}
}

// Set the server port.
func ServerWithInterface(self *Server, hostname string) {
	self.hostname = hostname
}

// Set the server port.
func ServerWithPort(self *Server, port int) {
	self.port = port
}

func ServerWithReadTimeout(self *Server, readTimeout time.Duration) {
	self.readTimeout = readTimeout
}

func ServerWithWriteTimeout(self *Server, writeTimeout time.Duration) {
	self.writeTimeout = writeTimeout
}

func ServerWithMaxHeaderBytes(self *Server, maxHeaderBytes int) {
	self.maxHeaderBytes = maxHeaderBytes
}

func ServerWithErrorLogger(self *Server, errorLogger *log.Logger) {
	self.errorLogger = errorLogger
}

func ServerWithTlsConfiguration(self *Server, tlsConfiguration *tls.Config) {
	self.tlsConfiguration = tlsConfiguration
}

// Start the server.
func ServerStart(self *Server) error {
	self.server = &http.Server{
		Handler:        self.mux,
		ReadTimeout:    self.readTimeout,
		WriteTimeout:   self.writeTimeout,
		MaxHeaderBytes: self.maxHeaderBytes,
		ErrorLog:       self.errorLogger,
		TLSConfig:      self.tlsConfiguration,
	}
	address := fmt.Sprintf("%s:%d", self.hostname, self.port)

	self.informationLogger.Printf("Listening for requests at http://%s", address)

	err := http.ListenAndServe(address, self.mux)
	if err != nil {
		return err
	}
	return nil
}

// Handle server requests.
func ServerOnRequest(
	self *Server,
	method string,
	path string,
	callback func(server *Server, request *Request, response *Response),
) {
	pattern := fmt.Sprintf("%s %s", strings.ToUpper(method), path)
	self.mux.HandleFunc(pattern, func(writer http.ResponseWriter, request *http.Request) {
		requestLocal := Request{
			server:  self,
			request: request,
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
	server  *Server
	request *http.Request
}

type Response struct {
	server                *Server
	writer                *http.ResponseWriter
	lockedStatusAndHeader bool
	statusCode            int
	header                *http.Header
}

// Handler server information.
func ServerOnInformation(self *Server, callback func(information string)) {
	self.informationHandler = append(self.informationHandler, callback)
}

// Handler server errors.
func ServerOnError(self *Server, callback func(err error)) {
	self.errorHandler = append(self.errorHandler, callback)
}

// Notify the server of some information.
func ServerNotifyInformation(self *Server, information string) {
	for _, listener := range self.informationHandler {
		listener(information)
	}
}

// Notify the server of an error.
func ServerNotifyError(self *Server, err error) {
	for _, listener := range self.errorHandler {
		listener(err)
	}
}

// Send the Status.
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
		ServerNotifyError(self.server, errors.New("Status is locked."))
		return
	}
	self.statusCode = code
}

// Send a Header.
//
// If the status has not been sent already, a default "200 OK" status will be sent immediately.
//
// This means the status will become locked and further attempts to send the status will fail with an error.
//
// You can retrieve this error using ServerOnError
func Header(self *Response, key string, value string) {
	if self.lockedStatusAndHeader {
		ServerNotifyError(self.server, errors.New("Headers locked."))
		return
	}

	self.header.Set(key, value)
}

// Send binary safe content.
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

// Send utf-8 safe content.
//
// Echo formats according to a format specifier.
//
// If the status code or the header have not been sent already, a default status of "200 OK" will be sent immediately along with whatever headers you've previously defined.
//
// The status code and the header will become locked and further attempts to send either of them will fail with an error.
//
// You can retrieve this error using ServerOnError.
//
// See fmt.Sprintf.
func Echo(self *Response, format string, a ...any) {
	Send(self, []byte(fmt.Sprintf(format, a...)))
}

func Accept(self *Request, acceptedMimes ...string) error {
	requestedMime := self.request.Header.Get("content-type")
	for _, acceptedMime := range acceptedMimes {
		if acceptedMime == "*" || strings.HasPrefix(requestedMime, acceptedMime) {
			return nil
		}
	}

	return fmt.Errorf("Requested mime type %s is not allowed.", requestedMime)
}
