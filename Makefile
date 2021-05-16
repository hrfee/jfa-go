GOESBUILD ?= off
ifeq ($(GOESBUILD), on)
	ESBUILD := esbuild
else
	ESBUILD := npx esbuild
endif
GOBINARY ?= go

VERSION ?= $(shell git describe --exact-match HEAD 2> /dev/null || echo vgit)
VERSION := $(shell echo $(VERSION) | sed 's/v//g')
COMMIT ?= $(shell git rev-parse --short HEAD || echo unknown)

UPDATER ?= off
LDFLAGS := -X main.version=$(VERSION) -X main.commit=$(COMMIT)
ifeq ($(UPDATER), on)
	LDFLAGS := $(LDFLAGS) -X main.updater=binary
else ifneq ($(UPDATER), off)
	LDFLAGS := $(LDFLAGS) -X main.updater=$(UPDATER)
endif

INTERNAL ?= on
ifeq ($(INTERNAL), on)
	TAGS :=
	DATA := data
else
	DATA := build/data
	TAGS := -tags external
endif

TRAY ?= off
ifeq ($(INTERNAL)$(TRAY), offon)
	TAGS := $(TAGS) tray
else ifeq ($(INTERNAL)$(TRAY), onon)
	TAGS := -tags tray
endif

OS := $(shell go env GOOS)
ifeq ($(TRAY)$(OS), onwindows)
	LDFLAGS := $(LDFLAGS) -H=windowsgui
endif

DEBUG ?= off
ifeq ($(DEBUG), on)
	SOURCEMAP := --sourcemap
	TYPECHECK := tsc -noEmit --project ts/tsconfig.json
	# jank
	COPYTS := rm -r $(DATA)/web/js/ts; cp -r ts $(DATA)/web/js
else
	LDFLAGS := -s -w $(LDFLAGS)
	SOURCEMAP :=
	COPYTS :=
	TYPECHECK :=
endif

npm:
	$(info installing npm dependencies)
	npm install
	@if [ "$(GOESBUILD)" = "off" ]; then\
		npm install esbuild;\
	else\
		go get -u github.com/evanw/esbuild/cmd/esbuild;\
	fi

configuration:
	$(info Fixing config-base)
	-mkdir -p $(DATA)
	python3 scripts/enumerate_config.py -i config/config-base.json -o $(DATA)/config-base.json
	$(info Generating config-default.ini)
	python3 scripts/generate_ini.py -i config/config-base.json -o $(DATA)/config-default.ini

email:
	$(info Generating email html)
	python3 scripts/compile_mjml.py -o $(DATA)/

typescript:
	$(TYPECHECK)
	$(info compiling typescript)
	-mkdir -p $(DATA)/web/js
	-$(ESBUILD) --bundle ts/admin.ts $(SOURCEMAP) --outfile=./$(DATA)/web/js/admin.js --minify
	-$(ESBUILD) --bundle ts/pwr.ts $(SOURCEMAP) --outfile=./$(DATA)/web/js/pwr.js --minify
	-$(ESBUILD) --bundle ts/form.ts $(SOURCEMAP) --outfile=./$(DATA)/web/js/form.js --minify
	-$(ESBUILD) --bundle ts/setup.ts $(SOURCEMAP) --outfile=./$(DATA)/web/js/setup.js --minify
	$(COPYTS)

swagger:
	$(GOBINARY) get github.com/swaggo/swag/cmd/swag
	swag init -g main.go

compile:
	$(info Downloading deps)
	$(GOBINARY) mod download
	$(info Building)
	mkdir -p build
	$(GOBINARY) build -ldflags="$(LDFLAGS)" $(TAGS) -o build/jfa-go

compress:
	upx --lzma build/jfa-go

bundle-css:
	-mkdir -p $(DATA)/web/css
	$(info bundling css)
	$(ESBUILD) --bundle css/base.css --outfile=$(DATA)/web/css/bundle.css --external:remixicon.css --minify

copy:
	$(info copying fonts)
	cp -r node_modules/remixicon/fonts/remixicon.css node_modules/remixicon/fonts/remixicon.woff2 $(DATA)/web/css/
	$(info copying html)
	cp -r html $(DATA)/
	$(info copying static data)
	-mkdir -p $(DATA)/web
	cp -r static/* $(DATA)/web/
	$(info copying systemd service)
	cp jfa-go.service $(DATA)/
	$(info copying language files)
	cp -r lang $(DATA)/
	cp LICENSE $(DATA)/

# internal-files:
# 	python3 scripts/embed.py internal
# 
# external-files:
# 	python3 scripts/embed.py external
# 	-mkdir -p build
# 	$(info copying internal data into build/)
# 	cp -r data build/

install:
	cp -r build $(DESTDIR)/jfa-go

clean:
	-rm -r $(DATA)
	-rm -r build
	-rm mail/*.html
	-rm docs/docs.go docs/swagger.json docs/swagger.yaml
	go clean

all: configuration npm email typescript bundle-css swagger copy compile
