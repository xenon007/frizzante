package guards

import frz "github.com/razshare/frizzante"

func GuardApi(_ *frz.Request, _ *frz.Response, pass func()) {
	// Guard.
	pass()
}

func GuardPages(_ *frz.Request, _ *frz.Response, p *frz.Page, pass func()) {
	// Guard.
	pass()
}
