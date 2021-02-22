module github.com/hrfee/jfa-go

go 1.16

replace github.com/hrfee/jfa-go/docs => ./docs

replace github.com/hrfee/jfa-go/mediabrowser => ./mediabrowser

replace github.com/hrfee/jfa-go/common => ./common

replace github.com/hrfee/jfa-go/ombi => ./ombi

require (
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/evanw/esbuild v0.8.50 // indirect
	github.com/fatih/color v1.10.0
	github.com/fsnotify/fsnotify v1.4.9
	github.com/gin-contrib/pprof v1.3.0
	github.com/gin-contrib/static v0.0.0-20200916080430-d45d9a37d28e
	github.com/gin-gonic/gin v1.6.3
	github.com/go-chi/chi v4.1.2+incompatible // indirect
	github.com/go-openapi/spec v0.20.3 // indirect
	github.com/go-playground/validator/v10 v10.4.1 // indirect
	github.com/golang/protobuf v1.4.3 // indirect
	github.com/gomarkdown/markdown v0.0.0-20210208175418-bda154fe17d8
	github.com/google/uuid v1.1.2 // indirect
	github.com/hrfee/jfa-go/common v0.0.0-20210105184019-fdc97b4e86cc
	github.com/hrfee/jfa-go/docs v0.0.0-20201112212552-b6f3cd7c1f71
	github.com/hrfee/jfa-go/mediabrowser v0.0.0-20201112212552-b6f3cd7c1f71
	github.com/hrfee/jfa-go/ombi v0.0.0-20201112212552-b6f3cd7c1f71
	github.com/jordan-wright/email v4.0.1-0.20200917010138-e1c00e156980+incompatible
	github.com/knz/strtime v0.0.0-20200924090105-187c67f2bf5e
	github.com/lithammer/shortuuid/v3 v3.0.4
	github.com/mailgun/mailgun-go/v4 v4.3.0
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/smartystreets/goconvey v1.6.4 // indirect
	github.com/swaggo/files v0.0.0-20190704085106-630677cd5c14
	github.com/swaggo/gin-swagger v1.3.0
	github.com/swaggo/swag v1.7.0 // indirect
	github.com/ugorji/go v1.2.0 // indirect
	github.com/writeas/go-strip-markdown v2.0.1+incompatible
	golang.org/x/crypto v0.0.0-20201112155050-0c6587e931a9 // indirect
	golang.org/x/sys v0.0.0-20210220050731-9a76102bfb43 // indirect
	golang.org/x/tools v0.1.0 // indirect
	google.golang.org/protobuf v1.25.0 // indirect
	gopkg.in/ini.v1 v1.62.0
)
