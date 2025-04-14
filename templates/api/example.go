package indexes

import f "github.com/razshare/frizzante"

func serveFunction(_ *f.Request, _ *f.Response) {
	// Serve api.
}

func Api(
	route f.RouteApiFunction,
	serve f.ServeApiFunction,
) {
	route("GET /")
	serve(serveFunction)
}
