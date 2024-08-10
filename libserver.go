package main

import (
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
)

type Server struct {
	serverErrorHandlers         []func(error)
	serverRequestErrorHandlers  []func(*Request, error)
	serverResponseErrorHandlers []func(*Request, *Response, error)
	serverResponseHandlers      []func(*Request, *Response) error
	sessions                    map[string]*net.Conn
	port                        int
	buffer                      []byte
}

// Create a Server.
func serverCreate(port int) *Server {
	return &Server{
		serverErrorHandlers:        []func(error){},
		serverRequestErrorHandlers: []func(*Request, error){},
		serverResponseHandlers:     []func(*Request, *Response) error{},
		port:                       port,
		buffer:                     make([]byte, 1024),
	}
}

// Start the Server.
func serverStart(self *Server) error {
	lis, err := net.Listen("tcp4", ":"+strconv.Itoa(self.port))
	if err != nil {
		return err
	}

	defer func(listener net.Listener) { _ = listener.Close() }(lis)
	for {
		con, err := lis.Accept()
		if err != nil {
			return err
		}
		serverRespond(self, con)
	}
}

// Handle incoming client socket.
func serverRespond(self *Server, socket net.Conn) {
	// Create the request.
	request := &Request{
		socket:  socket,
		headers: &Headers{},
		method:  "",
		path:    "",
		version: "",
	}

	// Find method.
	method, eol, methodError := requestFindNextWord(request)
	if methodError != nil {
		serverNotifyError(self, methodError)
		_ = socket.Close()
		return
	}

	// Make sure eol is not reached.
	if eol {
		serverNotifyRequestError(
			self, request, errors.New("request line must provide the method, path and protocol version before feeding a new line"),
		)
		_ = socket.Close()
		return
	}

	// Find path.
	path, eol, pathError := requestFindNextWord(request)
	if pathError != nil {
		serverNotifyError(self, pathError)
		_ = socket.Close()
		return
	}

	// Make sure eol is not reached.
	if eol {
		serverNotifyRequestError(
			self, request, errors.New("request line must provide a method, a path and the protocol version before feeding a new line"),
		)
		_ = socket.Close()
		return
	}

	// Find protocol.
	protocol, eol, protocolError := requestFindNextWord(request)
	if protocolError != nil {
		serverNotifyError(self, protocolError)
		_ = socket.Close()
		return
	}

	valueRawLength := len(protocol)
	if valueRawLength > 0 && cr == protocol[valueRawLength-1] {
		protocol = protocol[:valueRawLength-1]
	}

	// Make sure eol is reached.
	if !eol {
		serverNotifyRequestError(
			self, request, errors.New("request line must feed a new line after providing the method, path and protocol version"),
		)
		_ = socket.Close()
		return
	}

	// Find headers.
	headers := Headers{}
	for {
		key, eol, keyError := requestFindNextWord(request)
		if keyError != nil {
			serverNotifyError(self, keyError)
			_ = socket.Close()
			return
		}

		keyLength := len(key)

		// Check if eol is reached.
		if eol {
			// Check if it's the end of the headers section.
			if 0 == keyLength || (1 == keyLength && cr == key[0]) {
				// Happy path, we're done reading the headers, keep going.
				break
			}

			// Sad path, we just received a header key without a value.
			serverNotifyRequestError(
				self, request, errors.New("header lines must provide a key and a value before feeding a new line"),
			)
			_ = socket.Close()
			return
		}

		// Make sure the key name ends with a semicolon.
		if colon != key[keyLength-1] {
			keySyntaxError := errors.New("header field keys and values must be separated by `: ` (semicolon and one blank space)")
			serverNotifyRequestError(self, request, keySyntaxError)
			_ = socket.Close()
			return
		}

		// Strip the semicolon.
		keyStringified := strings.ToLower(string(key[:keyLength-1]))

		value, valueError := requestFindNextLine(request)
		if valueError != nil {
			return
		}

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
		socket:        socket,
		version:       string(protocol),
		lockedStatus:  false,
		lockedHeaders: false,
	}

	serverNotifyRequest(self, request, &resp)
	_ = socket.Close()
}

// Notify all listeners that a request has been received.
func serverNotifyRequest(self *Server, request *Request, response *Response) {
	for _, listener := range self.serverResponseHandlers {
		responseError := listener(request, response)
		if responseError != nil {
			serverNotifyResponseError(self, request, response, responseError)
		}
	}
}

// Register a callback to execute
// whenever the Server receives a Request.
func serverOnRequest(
	self *Server,
	method string,
	path string,
	callback func(*Request, *Response) error,
) {
	self.serverResponseHandlers = append(
		self.serverResponseHandlers, func(request *Request, response *Response) error {
			if method == request.method && path == request.path {
				err := callback(request, response)
				if err != nil {
					return err
				}
			}
			return nil
		},
	)
}

// Notify all listeners of a server error.
func serverNotifyError(self *Server, error error) {
	for _, listener := range self.serverErrorHandlers {
		listener(error)
	}
}

// Collect server errors.
func serverOnError(self *Server, callback func(error)) {
	self.serverErrorHandlers = append(self.serverErrorHandlers, callback)
}

// Notify all listeners of a request error.
func serverNotifyRequestError(self *Server, request *Request, error error) {
	for _, listener := range self.serverRequestErrorHandlers {
		listener(request, error)
	}
}

// Collect request errors.
func serverOnRequestError(self *Server, callback func(*Request, error)) {
	self.serverRequestErrorHandlers = append(self.serverRequestErrorHandlers, callback)
}

// Notify all listeners of a request error.
func serverNotifyResponseError(self *Server, request *Request, response *Response, error error) {
	for _, listener := range self.serverResponseErrorHandlers {
		listener(request, response, error)
	}
}

// Collect request errors.
func serverOnResponseError(self *Server, callback func(*Request, *Response, error)) {
	self.serverResponseErrorHandlers = append(self.serverResponseErrorHandlers, callback)
}

type Headers map[string]string

type Request struct {
	socket  net.Conn
	method  string
	path    string
	version string
	headers *Headers
}

// Find the next word in a request.
// This is useful when parsing headers in requests.
//
// A "word" is intended as any sequence of characters
// that doesn't contain blank spaces.
//
// This function will return the word (in bytes), a boolean indicating
// whether the word ended because of a new line and finally an error if found.
func requestFindNextWord(self *Request) ([]byte, bool, error) {
	reader := make([]byte, 1)
	var characters []byte
	read, readError := self.socket.Read(reader)
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

		read, readError = self.socket.Read(reader)
		if readError != nil {
			return nil, false, readError
		}
	}
	return characters, false, nil
}

func requestFindNextLine(self *Request) ([]byte, error) {
	var characters []byte
	reader := make([]byte, 1)
	read, readError := self.socket.Read(reader)

	if readError != nil {
		return nil, readError
	}

	for read > 0 {
		if eol == reader[0] {
			return characters, nil
		}

		characters = append(characters, reader[0])
		read, readError = self.socket.Read(reader)
		if readError != nil {
			return nil, readError
		}
	}
	return characters, nil
}

func requestWithMethod(self *Request, method string) {
	self.method = method
}

func requestWithPath(self *Request, path string) {
	self.path = path
}

func requestWithProtocol(self *Request, version string) {
	self.version = version
}

func requestWithHeaders(self *Request, headers *Headers) {
	self.headers = headers
}

type Response struct {
	socket        net.Conn
	version       string
	lockedStatus  bool
	lockedHeaders bool
}

// Send the status.
//
// This will lock the status, which makes it
// so that the next time you invoke this
// function it will return an error.
//
// The message defaults to blank.
//
// If the message is left blank,
// an algorithm will try to
// populate it automatically according to the status code.
func status(self *Response, status int, message string) error {
	if self.lockedStatus {
		return errors.New("status is locked")
	}
	messageLocal := message
	self.lockedStatus = true
	if "" == messageLocal {
		switch status {
		case 100:
			messageLocal = "Continue"
		case 101:
			messageLocal = "Switching Protocols"
		case 102:
			messageLocal = "Processing"
		case 103:
			messageLocal = "Early Hints"
		case 200:
			messageLocal = "OK"
		case 201:
			messageLocal = "Created"
		case 202:
			messageLocal = "Accepted"
		case 203:
			messageLocal = "Non-Authoritative Information"
		case 204:
			messageLocal = "No Content"
		case 205:
			messageLocal = "Reset Content"
		case 206:
			messageLocal = "Partial Content"
		case 207:
			messageLocal = "Multi-Status"
		case 208:
			messageLocal = "Already Reported"
		case 226:
			messageLocal = "IM Used"
		case 300:
			messageLocal = "Multiple Choices"
		case 301:
			messageLocal = "Moved Permanently"
		case 302:
			messageLocal = "Found"
		case 303:
			messageLocal = "See Other"
		case 304:
			messageLocal = "Not Modified"
		case 305:
			messageLocal = "Use Proxy"
		case 307:
			messageLocal = "Temporary Redirect"
		case 308:
			messageLocal = "Permanent Redirect"
		case 400:
			messageLocal = "Bad Request"
		case 401:
			messageLocal = "Unauthorized"
		case 402:
			messageLocal = "Payment Required"
		case 403:
			messageLocal = "Forbidden"
		case 404:
			messageLocal = "Not Found"
		case 405:
			messageLocal = "Method Not Allowed"
		case 406:
			messageLocal = "Not Acceptable"
		case 407:
			messageLocal = "Proxy Authentication Required"
		case 408:
			messageLocal = "Request Timeout"
		case 409:
			messageLocal = "Conflict"
		case 410:
			messageLocal = "Gone"
		case 411:
			messageLocal = "Length Required"
		case 412:
			messageLocal = "Precondition Failed"
		case 413:
			messageLocal = "Payload Too Large"
		case 414:
			messageLocal = "URI Too Long"
		case 415:
			messageLocal = "Unsupported Media Type"
		case 416:
			messageLocal = "Range Not Satisfiable"
		case 417:
			messageLocal = "Expectation Failed"
		case 421:
			messageLocal = "Misdirected Request"
		case 422:
			messageLocal = "Unprocessable Entity"
		case 423:
			messageLocal = "Locked"
		case 424:
			messageLocal = "Failed Dependency"
		case 426:
			messageLocal = "Upgrade Required"
		case 428:
			messageLocal = "Precondition Required"
		case 429:
			messageLocal = "Too Many Requests"
		case 431:
			messageLocal = "Request Header Fields Too Large"
		case 451:
			messageLocal = "Unavailable For Legal Reasons"
		case 500:
			messageLocal = "Internal Server Error"
		case 501:
			messageLocal = "Not Implemented"
		case 502:
			messageLocal = "Bad Gateway"
		case 503:
			messageLocal = "Service Unavailable"
		case 504:
			messageLocal = "Gateway Timeout"
		case 505:
			messageLocal = "HTTP Version Not Supported"
		case 506:
			messageLocal = "Variant Also Negotiates"
		case 507:
			messageLocal = "Insufficient Storage"
		case 508:
			messageLocal = "Loop Detected"
		case 510:
			messageLocal = "Not Extended"
		case 511:
			messageLocal = "Network Authentication Required"
		default:
			messageLocal = ""
		}
		if "" == messageLocal {
			return errors.New("unknown status code")
		}
	}
	feed := fmt.Sprintf("%s %d %s", self.version, status, message)
	_, err := self.socket.Write([]byte(feed))
	if err != nil {
		return err
	}
	return nil
}

// Send a header.
//
// If the status has not been sent already, it will be
// This means the status will become locked.
func header(self *Response, key string, value string) error {
	if !self.lockedStatus {
		err := status(self, 200, "OK")
		if err != nil {
			return err
		}
	}

	if self.lockedHeaders {
		return errors.New("headers locked")
	}

	_, err := self.socket.Write([]byte("\n" + key + ": " + value))
	if err != nil {
		return err
	}

	return nil
}

// Send binary safe content.
//
// If the status has not been sent already, it will be sent automatically as "200 OK".
// This means the status will become locked, meaning you can no longer set it after invoking this function.
//
// Headers will also be automatically locked.
func send(self *Response, value []byte) error {
	if !self.lockedStatus {
		err := status(self, 200, "OK")
		if err != nil {
			return err
		}
	}

	if !self.lockedHeaders {
		self.lockedHeaders = true
		_, err := self.socket.Write([]byte("\n\n"))
		if err != nil {
			return err
		}
	}

	_, err := self.socket.Write(value)
	if err != nil {
		return err
	}

	return nil
}

// Send utf8 content.
//
// If the status has not been sent already, it will be sent automatically as "200 OK".
// This means the status will become locked, meaning you can no longer set it after invoking this function.
//
// Headers will also be automatically locked.
func echo(self *Response, value string) error {
	return send(self, []byte(value))
}
