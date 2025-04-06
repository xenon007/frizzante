package indexes

import f "github.com/razshare/frizzante"

func indexShow(_ *f.Request, _ *f.Response, _ *f.Page) {
	// Show page.
}

func indexAction(_ *f.Request, _ *f.Response, _ *f.Page) {
	// Run page action.
}

func Index() (
	s f.PageFunction,
	a f.PageFunction,
) {
	s = indexShow
	a = indexAction
	return
}
