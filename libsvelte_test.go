package frizzante

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestRenderServer(test *testing.T) {
	server := ServerCreate()
	notifier := NotifierCreate()
	port := NextNumber(8080)
	ServerWithPort(server, port)
	ServerWithHostName(server, "127.0.0.1")
	ServerWithNotifier(server, notifier)
	ServerWithEmbeddedFileSystem(server, embeddedFileSystem)
	ServerWithPage(server, "/", "welcome",
		func() (
			show PageFunction,
			action PageFunction,
		) {
			show = func(req *Request, res *Response, p *Page) {
				PageWithRender(p, RenderServer)
				PageWithData(p, "name", "world")
			}
			return
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

func TestRenderClient(test *testing.T) {
	server := ServerCreate()
	notifier := NotifierCreate()
	port := NextNumber(8080)
	ServerWithPort(server, port)
	ServerWithNotifier(server, notifier)
	ServerWithHostName(server, "127.0.0.1")
	ServerWithEmbeddedFileSystem(server, embeddedFileSystem)
	ServerWithPage(server, "/", "welcome",
		func() (
			show PageFunction,
			action PageFunction,
		) {
			show = func(req *Request, res *Response, p *Page) {
				PageWithRender(p, RenderClient)
				PageWithData(p, "name", "world")
			}
			return
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
