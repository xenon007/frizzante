package main

import frz "github.com/razshare/frizzante"

func main() {
	frz.PrepareStart()
	frz.PrepareSveltePages("www/pages")
	frz.PrepareEnd()
}
