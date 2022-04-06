package main

import ()

var textOutputTemplate = `
################################################################################
                Jovian Decameter Radio Storm Forecast for:
                    {{.Start}}
                                until:
                    {{.End}}
{{if .Local}}                --- For coordinates {{.Lat}}º, {{.Lon}}º ---{{print "\n"}}{{end -}}
{{if .Location}}                Local time zone: {{.Location}} ({{.Offset}}){{print "\n"}}{{end -}}
################################################################################
{{.Data}}
################################################################################

`
