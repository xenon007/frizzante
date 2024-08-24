package main

func main() {
	server := ServerCreate()
	ServerWithPort(server, 8080)
	ServerOnResponseError(server, func(request *Request, response *Response, error error) {
		println("Response Error -", error.Error())
	})
	ServerOnInformation(server, func(information string) {
		println("Information -", information)
	})
	routerError := ServerWithFileSystemRouter(server, "router")
	if routerError != nil {
		println("Error -", routerError.Error())
		return
	}
	startError := ServerStart(server)
	if startError != nil {
		println(startError.Error())
		return
	}
}
