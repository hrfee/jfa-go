module github.com/hrfee/jfa-go

go 1.16

replace github.com/hrfee/jfa-go/docs => ./docs

replace github.com/hrfee/jfa-go/common => ./common

replace github.com/hrfee/jfa-go/ombi => ./ombi

replace github.com/hrfee/jfa-go/logger => ./logger

replace github.com/hrfee/jfa-go/linecache => ./linecache

replace github.com/hrfee/jfa-go/api => ./api

require (
	github.com/bwmarrin/discordgo v0.27.0
	github.com/emersion/go-autostart v0.0.0-20210130080809-00ed301c8e9a
	github.com/fatih/color v1.14.1
	github.com/fsnotify/fsnotify v1.6.0
	github.com/getlantern/errors v1.0.3 // indirect
	github.com/getlantern/golog v0.0.0-20230206140254-6d0a2e0f79af // indirect
	github.com/getlantern/hidden v0.0.0-20220104173330-f221c5a24770 // indirect
	github.com/getlantern/systray v1.2.1
	github.com/gin-contrib/pprof v1.4.0
	github.com/gin-contrib/static v0.0.1
	github.com/gin-gonic/gin v1.8.2
	github.com/go-openapi/jsonreference v0.20.2 // indirect
	github.com/go-openapi/spec v0.20.8 // indirect
	github.com/go-playground/validator/v10 v10.11.2 // indirect
	github.com/go-telegram-bot-api/telegram-bot-api v4.6.4+incompatible
	github.com/go-test/deep v1.1.0 // indirect
	github.com/goccy/go-json v0.10.0 // indirect
	github.com/golang-jwt/jwt v3.2.2+incompatible
	github.com/gomarkdown/markdown v0.0.0-20221013030248-663e2500819c
	github.com/google/uuid v1.3.0 // indirect
	github.com/hrfee/jfa-go/common v0.0.0-20230205010829-d56f321aad04
	github.com/hrfee/jfa-go/docs v0.0.0-20230205010829-d56f321aad04
	github.com/hrfee/jfa-go/linecache v0.0.0-20230205010829-d56f321aad04
	github.com/hrfee/jfa-go/logger v0.0.0-20230205010829-d56f321aad04
	github.com/hrfee/jfa-go/ombi v0.0.0-20230205010829-d56f321aad04
	github.com/hrfee/mediabrowser v0.3.8
	github.com/itchyny/timefmt-go v0.1.5
	github.com/lithammer/shortuuid/v3 v3.0.7
	github.com/mailgun/mailgun-go/v4 v4.8.2
	github.com/steambap/captcha v1.4.1
	github.com/swaggo/files v1.0.0
	github.com/swaggo/gin-swagger v1.5.3
	github.com/swaggo/swag v1.8.10 // indirect
	github.com/technoweenie/multipartstreamer v1.0.1 // indirect
	github.com/tidwall/pretty v1.2.1 // indirect
	github.com/ugorji/go/codec v1.2.9 // indirect
	github.com/writeas/go-strip-markdown v2.0.1+incompatible
	github.com/xhit/go-simple-mail/v2 v2.13.0
	go.opentelemetry.io/otel v1.13.0 // indirect
	go.uber.org/atomic v1.10.0 // indirect
	go.uber.org/multierr v1.9.0 // indirect
	go.uber.org/zap v1.24.0 // indirect
	golang.org/x/image v0.5.0 // indirect
	golang.org/x/net v0.7.0 // indirect
	golang.org/x/tools v0.6.0 // indirect
	gopkg.in/ini.v1 v1.67.0
	maunium.net/go/mautrix v0.14.0
)
