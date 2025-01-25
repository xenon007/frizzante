package frizzante

import (
	"rogchap.com/v8go"
	"strings"
	"testing"
	"time"
)

func TestEchoSveltePageModeServer(test *testing.T) {
	server := ServerCreate()
	ServerWithPort(server, 8084)
	ServerWithHostName(server, "127.0.0.1")
	ServerWithEmbeddedFileSystem(server, embeddedFileSystem)
	ServerOnError(server, func(err error) {
		test.Fatal(err)
	})
	ServerWithRequestHandler(server, "GET /", func(server *Server, request *Request, response *Response) {
		SendSveltePage(response, "welcome", &SveltePageConfiguration{
			Render: ModeServer,
			Data: map[string]interface{}{
				"name": "world",
			},
			Globals: map[string]v8go.FunctionCallback{},
		})
	})
	go ServerStart(server)
	time.Sleep(1 * time.Second)

	expected := "<h1>Hello world.</h1>"
	actual, getError := HttpGet("http://127.0.0.1:8084/", nil)
	if getError != nil {
		test.Fatal(getError)
	}

	ok := strings.Contains(actual, expected)

	if !ok {
		test.Fatalf("server was expected to respond with a string that contains '%s', received '%s' instead", expected, actual)
	}
}

func TestEchoSveltePageModeClient(test *testing.T) {
	server := ServerCreate()
	ServerWithPort(server, 8085)
	ServerWithHostName(server, "127.0.0.1")
	ServerWithEmbeddedFileSystem(server, embeddedFileSystem)
	ServerOnError(server, func(err error) {
		test.Fatal(err)
	})
	ServerWithRequestHandler(server, "GET /", func(server *Server, request *Request, response *Response) {
		SendSveltePage(response, "welcome", &SveltePageConfiguration{
			Render: ModeClient,
			Data: map[string]interface{}{
				"name": "world",
			},
			Globals: map[string]v8go.FunctionCallback{},
		})
	})
	go ServerStart(server)
	time.Sleep(1 * time.Second)

	expected := "<div id=\"app\"></div>"
	actual, getError := HttpGet("http://127.0.0.1:8085/", nil)
	if getError != nil {
		test.Fatal(getError)
	}

	ok := strings.Contains(actual, expected)

	if !ok {
		test.Fatalf("server was expected to respond with a string that contains '%s', received '%s' instead", expected, actual)
	}
}
