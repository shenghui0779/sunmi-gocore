// Code generated by hero.
// source: /Users/SM0286/code/core/gocore/tools/gocore/template/conf_base.got
// DO NOT EDIT!
package template

import "bytes"

func FromConfBase(buffer *bytes.Buffer) {
	buffer.WriteString(`
package conf

var baseConfig = ` + "`" + `
[network]
ApiServiceHost = ""
ApiServicePort = "80"
` + "`" + ``)

}