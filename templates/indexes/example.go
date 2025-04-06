package indexes

import f "github.com/razshare/frizzante"

func show(_ *f.Request, _ *f.Response, _ *f.Page) {
	// Show page.
}

func action(_ *f.Request, _ *f.Response, _ *f.Page) {
	// Run page action.
}

func index() (
	s f.PageFunction,
	a f.PageFunction,
) {
	s = show
	a = action
	return
}
