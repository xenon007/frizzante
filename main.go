package main

func main() {
	server := serverCreate(8080)

	serverOnRequest(
		server, "GET", "/", func(request *Request, response *Response) error {
			_ = header(response, "content-type", "text/plain")
			_ = echo(response, "hello world")
			return nil
		},
	)
	_ = serverStart(server)
}
