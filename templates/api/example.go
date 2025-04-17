package indexes

import f "github.com/razshare/frizzante"

func serveFunction(_ *f.Request, _ *f.Response) {
	// Serve api.
}

func Api(
	route func(pattern string),
	serve func(serveFunction func(req *f.Request, res *f.Response)),
) {
	route("GET /")
	serve(serveFunction)
}
