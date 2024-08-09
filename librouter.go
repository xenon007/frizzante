package main

type router struct {
	server *server
}

func router_create(server *server) *router {
	return &router{
		server: server,
	}
}

func router_map(self *router, method string, path string, callback func(*request, *response) error) {
	server_on_request(
		self.server, func(request *request, response *response) error {
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
