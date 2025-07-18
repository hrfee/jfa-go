module github.com/hrfee/jfa-go

go 1.23.0

toolchain go1.24.0

replace github.com/hrfee/jfa-go/docs => ./docs

replace github.com/hrfee/jfa-go/common => ./common

replace github.com/hrfee/jfa-go/ombi => ./ombi

replace github.com/hrfee/jfa-go/logger => ./logger

replace github.com/hrfee/jfa-go/logmessages => ./logmessages

replace github.com/hrfee/jfa-go/linecache => ./linecache

replace github.com/hrfee/jfa-go/api => ./api

replace github.com/hrfee/jfa-go/easyproxy => ./easyproxy

replace github.com/hrfee/jfa-go/jellyseerr => ./jellyseerr

require (
	github.com/bwmarrin/discordgo v0.29.0
	github.com/dgraph-io/badger/v4 v4.8.0
	github.com/emersion/go-autostart v0.0.0-20250403115856-34830d6457d2
	github.com/fatih/color v1.18.0
	github.com/fsnotify/fsnotify v1.9.0
	github.com/getlantern/systray v1.2.2
	github.com/gin-contrib/pprof v1.5.3
	github.com/gin-gonic/gin v1.10.1
	github.com/go-telegram-bot-api/telegram-bot-api v4.6.4+incompatible
	github.com/golang-jwt/jwt v3.2.2+incompatible
	github.com/gomarkdown/markdown v0.0.0-20250311123330-531bef5e742b
	github.com/hrfee/jfa-go/common v0.0.0-20250716174732-bcb6346f8115
	github.com/hrfee/jfa-go/docs v0.0.0-20250716174732-bcb6346f8115
	github.com/hrfee/jfa-go/easyproxy v0.0.0-20250716174732-bcb6346f8115
	github.com/hrfee/jfa-go/jellyseerr v0.0.0-20250716174732-bcb6346f8115
	github.com/hrfee/jfa-go/linecache v0.0.0-20250716174732-bcb6346f8115
	github.com/hrfee/jfa-go/logger v0.0.0-20250716174732-bcb6346f8115
	github.com/hrfee/jfa-go/logmessages v0.0.0-20250716174732-bcb6346f8115
	github.com/hrfee/jfa-go/ombi v0.0.0-20250716174732-bcb6346f8115
	github.com/hrfee/mediabrowser v0.3.29
	github.com/itchyny/timefmt-go v0.1.6
	github.com/lithammer/shortuuid/v3 v3.0.7
	github.com/mailgun/mailgun-go/v4 v4.23.0
	github.com/mattn/go-sqlite3 v1.14.28
	github.com/robert-nix/ansihtml v1.0.1
	github.com/steambap/captcha v1.4.1
	github.com/swaggo/files v1.0.1
	github.com/swaggo/gin-swagger v1.6.0
	github.com/timshannon/badgerhold/v4 v4.0.3
	github.com/writeas/go-strip-markdown v2.0.1+incompatible
	github.com/xhit/go-simple-mail/v2 v2.16.0
	gopkg.in/ini.v1 v1.67.0
	gopkg.in/yaml.v3 v3.0.1
	maunium.net/go/mautrix v0.24.2
)

require (
	filippo.io/edwards25519 v1.1.0 // indirect
	github.com/KyleBanks/depth v1.2.1 // indirect
	github.com/bytedance/sonic v1.13.3 // indirect
	github.com/bytedance/sonic/loader v0.3.0 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/cloudwego/base64x v0.1.5 // indirect
	github.com/cloudwego/iasm v0.2.0 // indirect
	github.com/dgraph-io/ristretto v1.0.0 // indirect
	github.com/dgraph-io/ristretto/v2 v2.2.0 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/gabriel-vasile/mimetype v1.4.9 // indirect
	github.com/getlantern/context v0.0.0-20220418194847-3d5e7a086201 // indirect
	github.com/getlantern/errors v1.0.4 // indirect
	github.com/getlantern/golog v0.0.0-20230503153817-8e72de7e0a65 // indirect
	github.com/getlantern/hex v0.0.0-20220104173244-ad7e4b9194dc // indirect
	github.com/getlantern/hidden v0.0.0-20220104173330-f221c5a24770 // indirect
	github.com/getlantern/ops v0.0.0-20231025133620-f368ab734534 // indirect
	github.com/gin-contrib/sse v1.1.0 // indirect
	github.com/go-chi/chi/v5 v5.2.2 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-openapi/jsonpointer v0.21.1 // indirect
	github.com/go-openapi/jsonreference v0.21.0 // indirect
	github.com/go-openapi/spec v0.21.0 // indirect
	github.com/go-openapi/swag v0.23.1 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/go-playground/validator/v10 v10.27.0 // indirect
	github.com/go-stack/stack v1.8.1 // indirect
	github.com/go-test/deep v1.1.0 // indirect
	github.com/goccy/go-json v0.10.5 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/freetype v0.0.0-20170609003504-e2365dfdc4a0 // indirect
	github.com/golang/groupcache v0.0.0-20241129210726-2c02b8208cf8 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/google/flatbuffers v25.2.10+incompatible // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/gorilla/websocket v1.5.3 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/klauspost/compress v1.18.0 // indirect
	github.com/klauspost/cpuid/v2 v2.3.0 // indirect
	github.com/leodido/go-urn v1.4.0 // indirect
	github.com/magisterquis/connectproxy v0.0.0-20200725203833-3582e84f0c9b // indirect
	github.com/mailgun/errors v0.4.0 // indirect
	github.com/mailru/easyjson v0.9.0 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/oxtoacart/bpool v0.0.0-20190530202638-03653db5a59c // indirect
	github.com/pelletier/go-toml/v2 v2.2.4 // indirect
	github.com/petermattis/goid v0.0.0-20250508124226-395b08cebbdb // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/rs/zerolog v1.34.0 // indirect
	github.com/sirupsen/logrus v1.9.3 // indirect
	github.com/swaggo/swag v1.16.4 // indirect
	github.com/technoweenie/multipartstreamer v1.0.1 // indirect
	github.com/tidwall/gjson v1.18.0 // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.1 // indirect
	github.com/tidwall/sjson v1.2.5 // indirect
	github.com/toorop/go-dkim v0.0.0-20250226130143-9025cce95817 // indirect
	github.com/twitchyliquid64/golang-asm v0.15.1 // indirect
	github.com/ugorji/go/codec v1.3.0 // indirect
	go.mau.fi/util v0.8.8 // indirect
	go.opencensus.io v0.24.0 // indirect
	go.opentelemetry.io/auto/sdk v1.1.0 // indirect
	go.opentelemetry.io/otel v1.37.0 // indirect
	go.opentelemetry.io/otel/metric v1.37.0 // indirect
	go.opentelemetry.io/otel/trace v1.37.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.27.0 // indirect
	golang.org/x/arch v0.19.0 // indirect
	golang.org/x/crypto v0.40.0 // indirect
	golang.org/x/exp v0.0.0-20250711185948-6ae5c78190dc // indirect
	golang.org/x/image v0.29.0 // indirect
	golang.org/x/net v0.42.0 // indirect
	golang.org/x/sys v0.34.0 // indirect
	golang.org/x/text v0.27.0 // indirect
	golang.org/x/tools v0.35.0 // indirect
	google.golang.org/protobuf v1.36.6 // indirect
)
