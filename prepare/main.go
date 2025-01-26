package main

import frz "github.com/razshare/frizzante"

func main() {
	frz.PrepareStart()
	frz.PrepareSveltePage("welcome", "www/lib/pages/welcome.svelte")
	frz.PrepareEnd()
}
