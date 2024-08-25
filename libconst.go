package frizzante

import (
	v8 "rogchap.com/v8go"
)

const eol = byte('\n')
const space = byte(' ')
const colon = byte(':')
const cr = byte('\r')
const lf = byte('\n')
const kb = 1024
const mb = 1024 * kb
const gb = 1024 * mb
const tb = 1024 * gb
const pb = 1024 * tb
const eb = 1024 * pb

const requirements = `
"use strict";
const performance = {
	now: function() {
	  return 0
	}
}
`

var boot = []byte("import { compile } from 'svelte/compiler'\n\nconst source = externGetArgs()\nconst compiled = compile(source, {generate: 'ssr'})\ncompiled.js.code")

var isolateGlobal = v8.NewIsolate()
var globalGlobal = v8.NewObjectTemplate(isolateGlobal)
var contextGlobal = v8.NewContext(isolateGlobal, globalGlobal)
