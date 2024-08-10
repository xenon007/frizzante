package main

func main() {
	server := serverCreate(8080)

	serverOnRequest(
		server, "GET", "/", func(request *Request, response *Response) error {
			println("Request received.")
			err := echo(response, "hello world")
			if err != nil {
				return err
			}
			return nil
		},
	)
	_ = serverStart(server)
}
