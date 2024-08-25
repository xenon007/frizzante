package frizzante

import (
	"crypto/tls"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Server struct {
	hostname             string
	port                 int
	server               *http.Server
	mux                  *http.ServeMux
	sessions             map[string]*net.Conn
	readTimeout          time.Duration
	writeTimeout         time.Duration
	maxHeaderBytes       int
	errorLogger          *log.Logger
	informationLogger    *log.Logger
	tlsConfiguration     *tls.Config
	informationHandler   []func(string)
	errorHandler         []func(error)
	temporaryDirectory   string
	nodeModulesDirectory string
}

// ServerCreate creates a server.
func ServerCreate() *Server {
	return &Server{
		hostname:             "",
		port:                 80,
		server:               nil,
		mux:                  http.NewServeMux(),
		sessions:             map[string]*net.Conn{},
		readTimeout:          10 * time.Second,
		writeTimeout:         10 * time.Second,
		maxHeaderBytes:       3 * mb,
		errorLogger:          log.Default(),
		informationLogger:    log.Default(),
		tlsConfiguration:     nil,
		informationHandler:   []func(string){},
		errorHandler:         []func(error){},
		temporaryDirectory:   ".temp",
		nodeModulesDirectory: "node_modules",
	}
}

// Set the server port.
func ServerWithInterface(self *Server, hostname string) {
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

// ServerWithTlsConfiguration sets the tls configuration.
func ServerWithTlsConfiguration(self *Server, tlsConfiguration *tls.Config) {
	self.tlsConfiguration = tlsConfiguration
}

// ServerWithTemporaryDirectory sets the tls configuration.
func ServerWithTemporaryDirectory(self *Server, temporaryDirectory string) {
	self.temporaryDirectory = temporaryDirectory
}

// ServerWithNodeModulesDirectory sets the tls configuration.
func ServerWithNodeModulesDirectory(self *Server, nodeModulesDirectory string) {
	self.nodeModulesDirectory = nodeModulesDirectory
}

// ServerWithFileServer creates a request handler that serves files from the local filesystem directories.
//
// Files ending with `.svelte` are compiled on the fly, cached, then served and reused for subsequent requests.
func ServerWithFileServer(self *Server, pattern string, directory string) {
	workspace := WorkspaceCreate()
	WorkspaceWithTemporaryDirectory(workspace, self.temporaryDirectory)
	WorkspaceWithNodeModulesDirectory(workspace, self.nodeModulesDirectory)

	cache := map[string]func(props map[string]any) (string, error){}

	//serveAssets := http.FileServer(http.Dir(strings.Join(directories, " ")))
	serveAssets := http.FileServer(http.Dir(directory))

	self.mux.HandleFunc(pattern, func(writer http.ResponseWriter, request *http.Request) {
		fileNameBase, cutError := strings.CutPrefix(request.RequestURI, "?")
		if !cutError {
			fileNameBase, cutError = strings.CutPrefix(request.RequestURI, "&")
		}

		fileNameAbsolute, absoluteFileNameError := filepath.Abs(fmt.Sprintf("%s%s", directory, fileNameBase))
		if nil != absoluteFileNameError {
			ServerNotifyError(self, absoluteFileNameError)
			return
		}

		if _, err := os.Stat(fileNameAbsolute); errors.Is(err, os.ErrNotExist) {
			// path/to/whatever does not exist
		}

		if strings.HasSuffix(fileNameAbsolute, ".svelte") {
			query, queryError := url.ParseQuery(request.RequestURI)
			if nil != queryError {
				ServerNotifyError(self, queryError)
				return
			}

			var html string
			var renderError error

			cachedRender, found := cache[fileNameAbsolute]
			if found {
				ServerNotifyInformation(self, fmt.Sprintf("Cache hit for svelte component `%s`.", fileNameAbsolute))
				html, renderError = cachedRender(map[string]any{
					"query": query,
				})
			} else {
				ServerNotifyInformation(self, fmt.Sprintf("Compiling svelte component `%s`...", fileNameAbsolute))
				render, compileError := WorkspaceCompileSvelte(workspace, fileNameAbsolute)
				cache[fileNameAbsolute] = render
				if compileError != nil {
					ServerNotifyError(self, compileError)
					return
				}
				ServerNotifyInformation(
					self, fmt.Sprintf("Svelte component `%s` has been compiled.", fileNameAbsolute),
				)
				html, renderError = render(map[string]any{
					"query": query,
				})
			}

			if nil != renderError {
				ServerNotifyError(self, renderError)
				return
			}

			header := writer.Header()

			response := &Response{
				server:                self,
				writer:                &writer,
				header:                &header,
				statusCode:            200,
				lockedStatusAndHeader: false,
			}
			Header(response, "content-type", "text/html")
			Echo(response, html)
			return
		}

		serveAssets.ServeHTTP(writer, request)
	})
}

// ServerStart starts the server.
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

// HandleFunc registers the handler function for the given pattern. If the given pattern conflicts, with one that is already registered, HandleFunc panics.
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

// ServerNotifyInformation notifies the server of some information.
func ServerLogInformation(self *Server, information string) {
	self.informationLogger.Println(information)
}

// ServerNotifyError notifies the server of an error.
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
		ServerNotifyError(self.server, errors.New("Status is locked."))
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
		ServerNotifyError(self.server, errors.New("Headers locked."))
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

// Accept returns an error if the incoming request does not specify a content-type header of accepted mimes.
func Accept(self *Request, acceptedMimes ...string) error {
	requestedMime := self.HttpRequest.Header.Get("content-type")
	for _, acceptedMime := range acceptedMimes {
		if acceptedMime == "*" || strings.HasPrefix(requestedMime, acceptedMime) {
			return nil
		}
	}

	return fmt.Errorf("Requested mime type %s is not allowed.", requestedMime)
}
