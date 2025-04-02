package frizzante

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestEchoSveltePageModeServer(test *testing.T) {
	server := ServerCreate()
	notifier := NotifierCreate()
	port := NextNumber(8080)
	ServerWithPort(server, port)
	ServerWithHostName(server, "127.0.0.1")
	ServerWithNotifier(server, notifier)
	ServerWithEmbeddedFileSystem(server, embeddedFileSystem)
	ServerRoutePage(server, "GET /", "Welcome",
		func(server *Server, request *Request, response *Response, page *Page) {
			PageWithRenderMode(page, RenderModeServer)
			PageWithData(page, "name", "world")
		},
	)
	go ServerStart(server)
	time.Sleep(1 * time.Second)

	expected := "<h1>Hello world.</h1>"
	actual, getError := HttpGet(fmt.Sprintf("http://127.0.0.1:%d/", port), nil)
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
	notifier := NotifierCreate()
	port := NextNumber(8080)
	ServerWithPort(server, port)
	ServerWithNotifier(server, notifier)
	ServerWithHostName(server, "127.0.0.1")
	ServerWithEmbeddedFileSystem(server, embeddedFileSystem)
	ServerRoutePage(server, "GET /", "Welcome",
		func(server *Server, request *Request, response *Response, page *Page) {
			PageWithRenderMode(page, RenderModeClient)
			PageWithData(page, "name", "world")
		},
	)
	go ServerStart(server)
	time.Sleep(1 * time.Second)

	expected := "<script type=\"application/javascript\">function target(){return document.getElementById("
	actual, getError := HttpGet(fmt.Sprintf("http://127.0.0.1:%d/", port), nil)
	if getError != nil {
		test.Fatal(getError)
	}

	ok := strings.Contains(actual, expected)

	if !ok {
		test.Fatalf("server was expected to respond with a string that contains '%s', received '%s' instead", expected, actual)
	}
}
