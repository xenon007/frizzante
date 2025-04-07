package indexes

import f "github.com/razshare/frizzante"

func indexShowFunction(_ *f.Request, _ *f.Response, _ *f.Page) {
	// Show page.
}

func indexActionFunction(_ *f.Request, _ *f.Response, _ *f.Page) {
	// Run page action.
}

func Index(
	route func(page string, path string),
	show func(showFunction f.PageFunction),
	action func(actionFunction f.PageFunction),
) {
	route("/path", "page")
	show(indexShowFunction)
	action(indexActionFunction)
}
