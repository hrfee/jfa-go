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
BUILDFLAGS := -X main.version=$(VERSION) -X main.commit=$(COMMIT)
ifeq ($(UPDATER), on)
	BUILDFLAGS := $(BUILDFLAGS) -X main.updater=binary
else ifneq ($(UPDATER), off)
	BUILDFLAGS := $(BUILDFLAGS) -X main.updater=$(UPDATER)
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
	-mkdir -p data
	python3 scripts/enumerate_config.py -i config/config-base.json -o data/config-base.json
	$(info Generating config-default.ini)
	python3 scripts/generate_ini.py -i config/config-base.json -o data/config-default.ini

email:
	$(info Generating email html)
	python3 scripts/compile_mjml.py -o data/

typescript:
	$(info compiling typescript)
	-mkdir -p data/web/js
	-$(ESBUILD) --bundle ts/admin.ts --outfile=./data/web/js/admin.js --minify
	-$(ESBUILD) --bundle ts/pwr.ts --outfile=./data/web/js/pwr.js --minify
	-$(ESBUILD) --bundle ts/form.ts --outfile=./data/web/js/form.js --minify
	-$(ESBUILD) --bundle ts/setup.ts --outfile=./data/web/js/setup.js --minify

ts-debug:
	$(info compiling typescript w/ sourcemaps)
	-mkdir -p data/web/js
	-$(ESBUILD) --bundle ts/admin.ts --sourcemap --outfile=./data/web/js/admin.js
	-$(ESBUILD) --bundle ts/pwr.ts --sourcemap --outfile=./data/web/js/pwr.js
	-$(ESBUILD) --bundle ts/form.ts --sourcemap --outfile=./data/web/js/form.js
	-$(ESBUILD) --bundle ts/setup.ts --sourcemap --outfile=./data/web/js/setup.js
	-rm -r data/web/js/ts
	$(info copying typescript)
	cp -r ts data/web/js

swagger:
	$(GOBINARY) get github.com/swaggo/swag/cmd/swag
	swag init -g main.go

compile:
	$(info Downloading deps)
	$(GOBINARY) mod download
	$(info Building)
	mkdir -p build
	cd build && CGO_ENABLED=0 $(GOBINARY) build -ldflags="-s -w $(BUILDFLAGS)" -o ./jfa-go ../*.go

compile-debug:
	$(info Downloading deps)
	$(GOBINARY) mod download
	$(info Building)
	mkdir -p build
	cd build && CGO_ENABLED=0 $(GOBINARY) build -ldflags "$(BUILDFLAGS)" -o ./jfa-go ../*.go

compress:
	upx --lzma build/jfa-go

bundle-css:
	-mkdir -p data/web/css
	$(info bundling css)
	$(ESBUILD) --bundle css/base.css --outfile=data/web/css/bundle.css --external:remixicon.css --minify

copy:
	$(info copying fonts)
	cp -r node_modules/remixicon/fonts/remixicon.css node_modules/remixicon/fonts/remixicon.woff2 data/web/css/
	$(info copying html)
	cp -r html data/
	$(info copying static data)
	-mkdir -p data/web
	cp -r static/* data/web/
	$(info copying language files)
	cp -r lang data/
	cp LICENSE data/

internal-files:
	python3 scripts/embed.py internal

external-files:
	python3 scripts/embed.py external
	-mkdir -p build
	$(info copying internal data into build/)
	cp -r data build/

install:
	cp -r build $(DESTDIR)/jfa-go

all: configuration npm email typescript bundle-css swagger copy internal-files compile
all-external: configuration npm email typescript bundle-css swagger copy external-files compile
debug: configuration npm email ts-debug bundle-css swagger copy external-files compile-debug
