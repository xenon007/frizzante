package frizzante

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestServerCreate(test *testing.T) {
	ServerCreate()
}

func TestServerWithHostName(test *testing.T) {
	server := ServerCreate()
	expected := "127.0.0.1"
	ServerWithHostName(server, expected)
	actual := server.hostName
	if actual != expected {
		test.Fatalf("server was expected to have host name '%s', received '%s' instead", expected, actual)
	}
}

func TestServerWithPort(test *testing.T) {
	server := ServerCreate()
	expected := 80
	ServerWithPort(server, expected)
	actual := server.port
	if actual != expected {
		test.Fatalf("server was expected to have port name %d, received %d instead", expected, actual)
	}
}

func TestServerWithReadTimeout(test *testing.T) {
	server := ServerCreate()
	expected := 10 * time.Second
	ServerWithReadTimeout(server, expected)
	actual := server.readTimeout
	if actual != expected {
		test.Fatalf("server was expected to have read timeout '%d', received '%d' instead", expected, actual)
	}

}

func TestServerWithWriteTimeout(test *testing.T) {
	server := ServerCreate()
	expected := 10 * time.Second
	ServerWithWriteTimeout(server, expected)
	actual := server.writeTimeout
	if actual != expected {
		test.Fatalf("server was expected to have write timeout '%d', received '%d' instead", expected, actual)
	}
}

func TestServerWithMaxHeaderBytes(test *testing.T) {
	server := ServerCreate()
	expected := 1 * MB
	ServerWithMaxHeaderBytes(server, expected)
	actual := server.maxHeaderBytes
	if actual != expected {
		test.Fatalf("server was expected to have max header bytes '%d', received '%d' instead", expected, actual)
	}
}

func TestServerWithErrorLogger(test *testing.T) {
	server := ServerCreate()
	expected := log.Default()
	ServerWithErrorLogger(server, expected)
	actual := server.errorLogger
	if actual != expected {
		test.Fatalf("server was expected to have default error logger")
	}
}

func TestServerWithInformationLogger(test *testing.T) {
	server := ServerCreate()
	expected := log.Default()
	ServerWithInformationLogger(server, expected)
	actual := server.informationLogger
	if actual != expected {
		test.Fatalf("server was expected to have default information logger")
	}
}

func TestServerWithCertificateAndKey(test *testing.T) {
	server := ServerCreate()
	expectedCertificate := "certificate.crt"
	expectedCertificateKey := "certificate.key"
	ServerWithCertificateAndKey(server, expectedCertificate, expectedCertificateKey)
	actualCertificate := server.certificate
	if actualCertificate != expectedCertificate {
		test.Fatalf("server was expected to have certificate '%s', received '%s' instead", expectedCertificate, actualCertificate)
	}
	actualCertificateKey := server.certificateKey
	if actualCertificateKey != expectedCertificateKey {
		test.Fatalf("server was expected to have certificate key '%s', received '%s' instead", expectedCertificateKey, actualCertificateKey)
	}
}

func TestServerWithTemporaryDirectory(test *testing.T) {
	server := ServerCreate()
	expected := ".temp"
	ServerWithTemporaryDirectory(server, expected)
	actual := server.temporaryDirectory
	if actual != expected {
		test.Fatalf("server was expected to have temporary directory '%s', received '%s' instead", expected, actual)
	}
}

func TestServerWithEmbeddedFileSystem(test *testing.T) {
	server := ServerCreate()
	expected := embeddedFileSystem
	ServerWithEmbeddedFileSystem(server, expected)
	actual := server.embeddedFileSystem
	if actual != expected {
		test.Fatalf("incorrect embedded file system detected")
	}
}

func TestServerSetTemporaryFile(test *testing.T) {
	server := ServerCreate()
	expected := "content"
	ServerWithTemporaryDirectory(server, ".temp")
	ServerSetTemporaryFile(server, "test", expected)
	fileName := filepath.Join(".temp", "test")
	if !Exists(fileName) {
		test.Fatalf("server was expected to create a temporary '%s', but it failed to do so", fileName)
	}
	bytes, readError := os.ReadFile(fileName)
	if readError != nil {
		test.Fatal(readError)
	}
	actual := string(bytes)
	if actual != expected {
		test.Fatalf("server temporary file was expected to contain '%s', received '%s' instead", expected, actual)
	}
}

func TestServerGetTemporaryFile(test *testing.T) {
	server := ServerCreate()
	expected := "content"
	ServerWithTemporaryDirectory(server, ".temp")
	ServerSetTemporaryFile(server, "test", expected)
	actual := ServerGetTemporaryFile(server, "test")
	if actual != expected {
		test.Fatalf("server temporary file was expected to contain '%s', received '%s' instead", expected, actual)
	}
}

func TestServerHasTemporaryFile(test *testing.T) {
	server := ServerCreate()
	ServerWithTemporaryDirectory(server, ".temp")
	ServerSetTemporaryFile(server, "test", "test")
	expected := true
	actual := ServerHasTemporaryFile(server, "test")
	if actual != expected {
		test.Fatalf("server was expected to have a temporary file by the name of 'test'")
	}
}

func TestServerClearTemporaryDirectory(test *testing.T) {
	server := ServerCreate()
	ServerWithTemporaryDirectory(server, ".temp")
	ServerSetTemporaryFile(server, "test", "test")
	ServerClearTemporaryDirectory(server)
	expected := false
	actual := ServerHasTemporaryFile(server, "test")
	if actual != expected {
		test.Fatalf("server was expected to not have a temporary file by the name of 'test'")
	}
}

func TestServerOnRequest(test *testing.T) {
	server := ServerCreate()
	ServerWithPort(server, 8080)
	expected := "hello"
	ServerOnRequest(server, "GET /", func(server *Server, request *Request, response *Response) {
		Echo(response, expected)
	})
	ServerOnError(server, func(err error) {
		test.Fatal(err)
	})
	go ServerStart(server)
	defer ServerStop(server)

	time.Sleep(1 * time.Second)

	actual, getError := HttpGet("http://127.0.0.1:8080/")
	if getError != nil {
		test.Fatal(getError)
	}

	if actual != expected {
		test.Fatalf("server was expected to respond with '%s', received '%s' instead", expected, actual)
	}
}

func TestServerOnInformation(test *testing.T) {
	actual := ""
	expected := "hello\nworld\n"
	server := ServerCreate()
	ServerOnInformation(server, func(information string) {
		actual += information + "\n"
	})
	ServerNotifyInformation(server, "hello")
	ServerNotifyInformation(server, "world")

	if actual != expected {
		test.Fatalf("server was expected to log informations '%s', received '%s' instead", expected, actual)
	}
}

func TestServerOnError(test *testing.T) {
	actual := ""
	expected := "hello\nworld\n"
	server := ServerCreate()
	ServerOnError(server, func(err error) {
		actual += err.Error() + "\n"
	})
	ServerNotifyError(server, fmt.Errorf("hello"))
	ServerNotifyError(server, fmt.Errorf("world"))

	if actual != expected {
		test.Fatalf("server was expected to log errors '%s', received '%s' instead", expected, actual)
	}
}

func TestStatus(test *testing.T) {
	server := ServerCreate()
	ServerWithPort(server, 8081)
	expected := 201
	ServerOnRequest(server, "GET /", func(server *Server, request *Request, response *Response) {
		Status(response, expected)
		Echo(response, "Ok")
	})
	ServerOnError(server, func(err error) {
		test.Fatal(err)
	})
	go ServerStart(server)
	defer ServerStop(server)

	time.Sleep(1 * time.Second)

	response, getError := http.Get("http://127.0.0.1:8081/")
	if getError != nil {
		test.Fatal(getError)
	}
	defer response.Body.Close()

	actual := response.StatusCode

	if actual != expected {
		test.Fatalf("server was expected to respond with status code '%d', received '%d' intead", expected, actual)
	}
}

func TestHeader(test *testing.T) {
	server := ServerCreate()
	ServerWithPort(server, 8082)
	expected := "application/json"
	ServerOnRequest(server, "GET /", func(server *Server, request *Request, response *Response) {
		Header(response, "Content-Type", expected)
		Echo(response, "{}")
	})
	ServerOnError(server, func(err error) {
		test.Fatal(err)
	})
	go ServerStart(server)
	defer ServerStop(server)

	time.Sleep(1 * time.Second)

	response, getError := http.Get("http://127.0.0.1:8082/")
	if getError != nil {
		test.Fatal(getError)
	}
	defer response.Body.Close()

	actual := response.Header.Get("Content-Type")

	if actual != expected {
		test.Fatalf("server was expected to respond with header content type '%s', received '%s' intead", expected, actual)
	}
}

//func TestSend(test *testing.T) {
//}
//
//func TestEcho(test *testing.T) {
//}
//
//func TestAccept(test *testing.T) {
//}
//
//func TestEmbeddedFileOrElse(test *testing.T) {
//}
//
//func TestEmbeddedFileOrIndexElse(test *testing.T) {
//}
//
//func TestFileOrIndexElse(test *testing.T) {
//}
//
//func TestFileOrElse(test *testing.T) {
//}
