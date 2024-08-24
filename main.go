package main

func main() {
	server := ServerCreate(8080)

	ServerOnResponseError(server, func(request *Request, response *Response, err error) {
		println("Response Error -", err.Error())
	})

	ServerOnRequest(
		server, "POST", "/", func(request *Request, response *Response) {
			form, _ := Form(request)
			username := form.Value["username"][0]
			Header(response, "content-type", "text/html")
			Status(response, 200)
			Echo(response, "hello %s", username)
		},
	)
	_ = ServerStart(server)
}
