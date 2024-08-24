package main

import (
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"plugin"
	"strconv"
	"strings"
)

type Server struct {
	ServerInformationHandlers   []func(string)
	ServerErrorHandlers         []func(error)
	ServerRequestErrorHandlers  []func(*Request, error)
	ServerResponseErrorHandlers []func(*Request, *Response, error)
	ServerResponseHandlers      []func(*Request, *Response)
	Sessions                    map[string]*net.Conn
	Port                        int
	Buffer                      []byte
}

// Create a server.
func ServerCreate() *Server {
	return &Server{
		ServerInformationHandlers:  []func(string){},
		ServerErrorHandlers:        []func(error){},
		ServerRequestErrorHandlers: []func(*Request, error){},
		ServerResponseHandlers:     []func(*Request, *Response){},
		Port:                       80,
	}
}

// Set the server port.
func ServerWithPort(self *Server, port int) {
	self.Port = port
}

// Start the server.
func ServerStart(self *Server) error {
	listener, listenError := net.Listen("tcp4", ":"+strconv.Itoa(self.Port))
	if listenError != nil {
		return listenError
	}

	ServerNotifyInformation(self, fmt.Sprintf("Listening on http://127.0.0.1:%d", self.Port))

	defer func(listener net.Listener) { _ = listener.Close() }(listener)
	for {
		acceptedConnection, acceptError := listener.Accept()
		if acceptError != nil {
			return acceptError
		}
		ServerRespond(self, &acceptedConnection)
	}
}

// Handle incoming client Socket.
func ServerRespond(self *Server, socket *net.Conn) {
	// Create the request.
	request := &Request{
		Socket:  socket,
		Server:  self,
		Headers: &Headers{},
		Method:  "",
		Path:    "",
		Version: "",
	}

	// Find Method.
	method, eol, methodError := requestFindNextWord(request)
	if methodError != nil {
		ServerNotifyError(self, methodError)
		_ = (*socket).Close()
		return
	}

	// Make sure eol is not reached.
	if eol {
		ServerNotifyRequestError(
			self, request, errors.New("request line must provide the Method, Path and protocol Version before feeding a new line"),
		)
		_ = (*socket).Close()
		return
	}

	// Find Path.
	path, eol, pathError := requestFindNextWord(request)
	if pathError != nil {
		ServerNotifyError(self, pathError)
		_ = (*socket).Close()
		return
	}

	// Make sure eol is not reached.
	if eol {
		ServerNotifyRequestError(
			self, request, errors.New("request line must provide a Method, a Path and the protocol Version before feeding a new line"),
		)
		_ = (*socket).Close()
		return
	}

	// Find protocol.
	protocol, eol, protocolError := requestFindNextWord(request)
	if protocolError != nil {
		ServerNotifyError(self, protocolError)
		_ = (*socket).Close()
		return
	}

	valueRawLength := len(protocol)
	if valueRawLength > 0 && cr == protocol[valueRawLength-1] {
		protocol = protocol[:valueRawLength-1]
	}

	// Make sure eol is reached.
	if !eol {
		ServerNotifyRequestError(
			self, request, errors.New("request line must feed a new line after providing the Method, Path and protocol Version"),
		)
		_ = (*socket).Close()
		return
	}

	// Find Headers.
	headers := Headers{}
	for {
		key, eol, keyError := requestFindNextWord(request)
		if keyError != nil {
			ServerNotifyError(self, keyError)
			_ = (*socket).Close()
			return
		}

		keyLength := len(key)

		// Check if eol is reached.
		if eol {
			// Check if it's the end of the Headers section.
			if 0 == keyLength || (1 == keyLength && cr == key[0]) {
				// Happy Path, we're done reading the Headers, keep going.
				break
			}

			// Sad Path, we just received a Header key without a value.
			ServerNotifyRequestError(
				self, request, errors.New("Header lines must provide a key and a value before feeding a new line"),
			)
			_ = (*socket).Close()
			return
		}

		// Make sure the key name ends with a semicolon.
		if colon != key[keyLength-1] {
			keySyntaxError := errors.New("Header field keys and values must be separated by `: ` (semicolon and one blank space)")
			ServerNotifyRequestError(self, request, keySyntaxError)
			_ = (*socket).Close()
			return
		}

		// Strip the semicolon.
		keyStringified := strings.ToLower(string(key[:keyLength-1]))

		value := requestFindNextLine(request)

		// Strip \r from the end of the string.
		valueLength := len(value)
		if valueLength > 0 && cr == value[valueLength-1] {
			value = value[:valueLength-1]
		}

		headers[keyStringified] = string(value)
	}

	requestWithMethod(request, string(method))
	requestWithPath(request, string(path))
	requestWithProtocol(request, string(protocol))
	requestWithHeaders(request, &headers)

	resp := Response{
		Socket:        socket,
		Server:        self,
		Request:       request,
		Version:       string(protocol),
		LockedStatus:  false,
		LockedHeaders: false,
	}

	ServerNotifyRequest(self, request, &resp)
	_ = (*socket).Close()
}

// Notify all listeners that a request has been received.
func ServerNotifyRequest(self *Server, request *Request, response *Response) {
	for _, listener := range self.ServerResponseHandlers {
		listener(request, response)
	}
}

// Register a callback to execute
// whenever the server receives a Request.
func ServerOnRequest(
	self *Server,
	method string,
	path string,
	callback func(request *Request, response *Response),
) {
	self.ServerResponseHandlers = append(
		self.ServerResponseHandlers, func(request *Request, response *Response) {
			if method == request.Method && path == request.Path {
				callback(request, response)
			}
		},
	)
}

// Use the filesystem as a router.
func ServerWithFileSystemRouter(self *Server, directory string) {
	soIdentifier := "frizzante.so"
	soIdentifierLength := len(soIdentifier)
	ServerNotifyInformation(self, fmt.Sprintf("Walking directory `%s`", directory))
	walkError := filepath.Walk(
		directory, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				ServerNotifyError(self, err)
				return nil
			}

			if !strings.HasSuffix(path, soIdentifier) {
				return nil
			}

			ServerNotifyInformation(self, fmt.Sprintf("Router plugin found `%s`", path))

			filePath := strings.Replace(path, directory, "", 1)[soIdentifierLength-2:]
			filePathChunks := strings.Split(filePath, "/")
			filePathChunksLength := len(filePathChunks)
			webPath := strings.Join(filePathChunks[:filePathChunksLength-1], "/")

			if !strings.HasPrefix(webPath, "/") {
				webPath = "/" + webPath
			}

			ServerNotifyInformation(self, fmt.Sprintf("Attempting to load plugin `%s`...", path))

			pluginLocal, pluginError := plugin.Open(path)
			if pluginError != nil {
				ServerNotifyError(self, pluginError)
				return nil
			}

			ServerNotifyInformation(self, fmt.Sprintf("Plugin `%s` loaded.", path))

			webFunctions := map[string]func(*Request, *Response){}

			symbol, err := pluginLocal.Lookup("Get")
			if err == nil {
				webFunctions["Get"] = symbol.(func(*Request, *Response))
			}

			for webMethod, webFunction := range webFunctions {
				ServerNotifyInformation(self, fmt.Sprintf("Listening for requests at %s %s", webMethod, webPath))
				ServerOnRequest(self, webMethod, webPath, webFunction)
				ServerOnRequest(self, webMethod, webPath+"/", func(request *Request, response *Response) {
					Status(response, 308)
					Header(response, "Location", webPath)
				})
			}
			return nil
		},
	)

	if walkError != nil {
		ServerNotifyError(self, walkError)
		return
	}
}

// Notify all listeners of a server error.
func ServerNotifyInformation(self *Server, information string) {
	for _, listener := range self.ServerInformationHandlers {
		listener(information)
	}
}

// Notify all listeners of a server error.
func ServerNotifyError(self *Server, err error) {
	for _, listener := range self.ServerErrorHandlers {
		listener(err)
	}
}

// Collect server information.
func ServerOnInformation(self *Server, callback func(information string)) {
	self.ServerInformationHandlers = append(self.ServerInformationHandlers, callback)
}

// Collect server errors.
func ServerOnError(self *Server, callback func(err error)) {
	self.ServerErrorHandlers = append(self.ServerErrorHandlers, callback)
}

// Notify all listeners of a request error.
func ServerNotifyRequestError(self *Server, request *Request, err error) {
	for _, listener := range self.ServerRequestErrorHandlers {
		listener(request, err)
	}
}

// Collect request errors.
func ServerOnRequestError(self *Server, callback func(request *Request, err error)) {
	self.ServerRequestErrorHandlers = append(self.ServerRequestErrorHandlers, callback)
}

// Notify all listeners of a request error.
func ServerNotifyResponseError(self *Server, request *Request, response *Response, err error) {
	for _, listener := range self.ServerResponseErrorHandlers {
		listener(request, response, err)
	}
}

// Collect request errors.
func ServerOnResponseError(self *Server, callback func(request *Request, response *Response, err error)) {
	self.ServerResponseErrorHandlers = append(self.ServerResponseErrorHandlers, callback)
}

type Headers map[string]string

type Request struct {
	Socket  *net.Conn
	Server  *Server
	Method  string
	Path    string
	Version string
	Headers *Headers
}

// Find the next word in a request.
// This is useful when parsing Headers in requests.
//
// A "word" is intended as any sequence of characters
// that doesn't contain blank spaces.
//
// This function will return the word (in bytes), a boolean indicating
// whether the word ended because of a new line and finally an error if found.
func requestFindNextWord(self *Request) ([]byte, bool, error) {
	reader := make([]byte, 1)
	var characters []byte
	socket := *self.Socket
	read, readError := socket.Read(reader)
	if readError != nil {
		return nil, false, readError
	}

	for read > 0 {
		if space == reader[0] {
			return characters, false, nil
		} else if lf == reader[0] {
			return characters, true, nil
		}

		characters = append(characters, reader[0])

		read, readError = socket.Read(reader)
		if readError != nil {
			return nil, false, readError
		}
	}
	return characters, false, nil
}

// Find the next line in the Request.
func requestFindNextLine(self *Request) []byte {
	var characters []byte
	reader := make([]byte, 1)
	socket := *self.Socket
	read, readError := socket.Read(reader)

	if readError != nil {
		return characters
	}

	for read > 0 {
		if eol == reader[0] {
			return characters
		}

		characters = append(characters, reader[0])
		read, readError = socket.Read(reader)
		if readError != nil {
			return characters
		}
	}
	return characters
}

// Set the method of the Request.
func requestWithMethod(self *Request, method string) {
	self.Method = method
}

// Set the path of the Request.
func requestWithPath(self *Request, path string) {
	self.Path = path
}

// Set the protocol of the Request.
func requestWithProtocol(self *Request, version string) {
	self.Version = version
}

// Set the Headers of the Request.
func requestWithHeaders(self *Request, headers *Headers) {
	self.Headers = headers
}

// Parse the body of the Request as form.
// Supports multipart and url encodings.
func Form(self *Request) (*multipart.Form, error) {
	headers := *self.Headers

	contentType := headers["content-type"]
	rootMime := "text/plain"
	boundary := ""

	for index, item := range strings.Split(contentType, "; ") {
		if 0 == index {
			rootMime = item
			continue
		}

		attribute := strings.Split(item, "=")
		attributeLength := len(attribute)
		if attributeLength != 2 || "boundary" != attribute[0] {
			return nil, fmt.Errorf("Multipart boundary identifier should follow the mime type in content-type as `boundary=some_identifier`, provided `%s` instead.", item)
		}
		boundary = attribute[1]
	}

	if "application/x-www-form-urlencoded" == rootMime {
		line := requestFindNextLine(self)
		query, err := url.ParseQuery(string(line))
		if err != nil {
			return nil, err
		}

		form := multipart.Form{}

		for key, value := range query {
			form.Value[key] = value
		}
		return &form, nil
	} else if "" == boundary {
		return nil, errors.New("Invalid empty boundary provided.")
	}

	if "multipart/form-data" != rootMime && "multipart/mixed" != rootMime && "multipart/alternative" != rootMime && "text/plain" != rootMime {
		return nil, fmt.Errorf("Multipart requests accept mimes of `multipart/form-data`, `multipart/mixed`, `multipart/alternative`, `text/plain` and nothing else, provided `%s` instead.", rootMime)
	}

	var rio io.Reader = &Reader{Request: self}

	reader := multipart.NewReader(rio, boundary)

	form, err := reader.ReadForm(2 * gb)
	if err != nil {
		return nil, err
	}

	return form, nil
}

type Reader struct {
	Request *Request
}

func (r *Reader) Read(p []byte) (n int, err error) {
	socket := *r.Request.Socket

	read, err := socket.Read(p)
	if err != nil {
		return 0, err
	}

	return read, nil
}

type Response struct {
	Socket        *net.Conn
	Server        *Server
	Request       *Request
	Version       string
	LockedStatus  bool
	LockedHeaders bool
}

// Send the Status.
//
// The status message will be inferred automatically based on the code.
//
// This will lock the status, which makes it
// so that the next time you invoke this
// function it will fail with an error.
//
// You can retrieve this error using ServerOnResponseError.
func Status(self *Response, code int) {
	var message string
	switch code {
	case 100:
		message = "Continue"
	case 101:
		message = "Switching Protocols"
	case 102:
		message = "Processing"
	case 103:
		message = "Early Hints"
	case 200:
		message = "OK"
	case 201:
		message = "Created"
	case 202:
		message = "Accepted"
	case 203:
		message = "Non-Authoritative Information"
	case 204:
		message = "No Content"
	case 205:
		message = "Reset Content"
	case 206:
		message = "Partial Content"
	case 207:
		message = "Multi-Status"
	case 208:
		message = "Already Reported"
	case 226:
		message = "IM Used"
	case 300:
		message = "Multiple Choices"
	case 301:
		message = "Moved Permanently"
	case 302:
		message = "Found"
	case 303:
		message = "See Other"
	case 304:
		message = "Not Modified"
	case 305:
		message = "Use Proxy"
	case 307:
		message = "Temporary Redirect"
	case 308:
		message = "Permanent Redirect"
	case 400:
		message = "Bad Request"
	case 401:
		message = "Unauthorized"
	case 402:
		message = "Payment Required"
	case 403:
		message = "Forbidden"
	case 404:
		message = "Not Found"
	case 405:
		message = "Method Not Allowed"
	case 406:
		message = "Not Acceptable"
	case 407:
		message = "Proxy Authentication Required"
	case 408:
		message = "Request Timeout"
	case 409:
		message = "Conflict"
	case 410:
		message = "Gone"
	case 411:
		message = "Length Required"
	case 412:
		message = "Precondition Failed"
	case 413:
		message = "Payload Too Large"
	case 414:
		message = "URI Too Long"
	case 415:
		message = "Unsupported Media Type"
	case 416:
		message = "Range Not Satisfiable"
	case 417:
		message = "Expectation Failed"
	case 421:
		message = "Misdirected Request"
	case 422:
		message = "Unprocessable Entity"
	case 423:
		message = "Locked"
	case 424:
		message = "Failed Dependency"
	case 426:
		message = "Upgrade Required"
	case 428:
		message = "Precondition Required"
	case 429:
		message = "Too Many Requests"
	case 431:
		message = "Request Header Fields Too Large"
	case 451:
		message = "Unavailable For Legal Reasons"
	case 500:
		message = "Internal server Error"
	case 501:
		message = "Not Implemented"
	case 502:
		message = "Bad Gateway"
	case 503:
		message = "Service Unavailable"
	case 504:
		message = "Gateway Timeout"
	case 505:
		message = "HTTP Version Not Supported"
	case 506:
		message = "Variant Also Negotiates"
	case 507:
		message = "Insufficient Storage"
	case 508:
		message = "Loop Detected"
	case 510:
		message = "Not Extended"
	case 511:
		message = "Network Authentication Required"
	default:
		message = ""
	}
	if "" == message {
		ServerNotifyResponseError(self.Server, self.Request, self, errors.New("unknown Status code"))
		return
	}
	StatusMessage(self, code, message)
}

// Send the status code and message.
//
// This will lock the status, which makes it
// so that the next time you invoke this
// function it will fail with an error.
//
// You can retrieve this error using ServerOnResponseError.
func StatusMessage(self *Response, code int, message string) {
	if self.LockedStatus {
		ServerNotifyResponseError(self.Server, self.Request, self, errors.New("Status is locked"))
		return
	}
	self.LockedStatus = true
	feed := fmt.Sprintf("%s %d %s", self.Version, code, message)
	_, err := (*self.Socket).Write([]byte(feed))
	if err != nil {
		ServerNotifyResponseError(self.Server, self.Request, self, err)
		return
	}
}

// Send a Header.
//
// If the status has not been sent already, a default "200 OK" status will be sent immediately.
//
// This means the status will become locked and further attempts to send the status will fail with an error.
//
// You can retrieve this error using ServerOnResponseError.
func Header(self *Response, key string, value string) {
	if !self.LockedStatus {
		StatusMessage(self, 200, "OK")
	}

	if self.LockedHeaders {
		ServerNotifyResponseError(self.Server, self.Request, self, errors.New("Headers locked"))
		return
	}

	_, err := (*self.Socket).Write([]byte("\r\n" + key + ": " + value))
	if err != nil {
		ServerNotifyResponseError(self.Server, self.Request, self, err)
		return
	}
}

// Send binary safe content.
//
// If the status has not been sent already, a default "200 OK" status will be sent immediately.
//
// This means the status will become locked and further attempts to send the status will fail with an error.
//
// Headers will also be automatically locked and further attempts to send headers will fail with errors.
//
// You can retrieve these errors using ServerOnResponseError.
func Send(self *Response, value []byte) {
	if !self.LockedStatus {
		StatusMessage(self, 200, "OK")
	}

	socket := *self.Socket

	if !self.LockedHeaders {
		self.LockedHeaders = true
		_, err := socket.Write([]byte("\r\n\r\n"))
		if err != nil {
			ServerNotifyResponseError(self.Server, self.Request, self, err)
			return
		}
	}

	_, err := socket.Write(value)
	if err != nil {
		ServerNotifyResponseError(self.Server, self.Request, self, err)
		return
	}
}

// Send utf-8 safe content.
//
// Echo formats according to a format specifier.
//
// If the status has not been sent already, a default "200 OK" status will be sent immediately.
//
// This means the status will become locked and further attempts to send the status will fail with an error.
//
// Headers will also be automatically locked and further attempts to send headers will fail with errors.
//
// You can retrieve these errors using ServerOnResponseError.
//
// See fmt.Sprintf.
func Echo(self *Response, format string, a ...any) {
	Send(self, []byte(fmt.Sprintf(format, a...)))
}

func Accept(self *Request, acceptedMimes ...string) error {
	requestedMime := (*self.Headers)["content-type"]
	for _, acceptedMime := range acceptedMimes {
		if acceptedMime == "*" || strings.HasPrefix(requestedMime, acceptedMime) {
			return nil
		}
	}

	return fmt.Errorf("Requested mime type %s is not allowed.", requestedMime)
}

func POST(self *Request, key string) {

}
