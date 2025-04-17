package indexes

import f "github.com/razshare/frizzante"

func indexShowFunction(_ *f.Request, _ *f.Response, _ *f.Page) {
	// Show page.
}

func indexActionFunction(_ *f.Request, _ *f.Response, _ *f.Page) {
	// Run page action.
}

func Index(
	route func(path string, page string),
	show func(showFunction func(req *f.Request, res *f.Response, p *f.Page)),
	action func(actionFunction func(req *f.Request, res *f.Response, p *f.Page)),
) {
	route("/path", "page")
	show(indexShowFunction)
	action(indexActionFunction)
}
