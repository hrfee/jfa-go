module github.com/hrfee/jfa-go/scripts/ini

replace github.com/hrfee/jfa-go/common => ../../common

replace github.com/hrfee/jfa-go/logmessages => ../../logmessages

go 1.22.4

require (
	github.com/fatih/color v1.18.0
	github.com/hrfee/jfa-go/common v0.0.0-00010101000000-000000000000
	gopkg.in/ini.v1 v1.67.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/hrfee/jfa-go/logmessages v0.0.0-20240806200606-6308db495a0a // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/stretchr/testify v1.11.1 // indirect
	golang.org/x/sys v0.25.0 // indirect
)
