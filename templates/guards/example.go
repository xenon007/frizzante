package guards

import frz "github.com/razshare/frizzante"

func guardApi(_ *frz.Request, _ *frz.Response, pass func()) {
	// Guard.
	pass()
}

func guardPages(_ *frz.Request, _ *frz.Response, p *frz.Page, pass func()) {
	// Guard.
	pass()
}
