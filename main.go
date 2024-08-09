package main

func main() {
	s := server_create(8080)
	r := router_create(s)
	router_map(
		r, "GET", "/", func(request *request, response *response) error {
			println("Request received.")
			err := echo(response, "hello world")
			if err != nil {
				return err
			}
			return nil
		},
	)
	err := server_start(s)
	if err != nil {
		println(err)
	}
}
