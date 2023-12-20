GOESBUILD ?= off
ifeq ($(GOESBUILD), on)
	ESBUILD := esbuild
else
	ESBUILD := npx esbuild
endif
GOBINARY ?= go

CSSVERSION ?= v3

VERSION ?= $(shell git describe --exact-match HEAD 2> /dev/null || echo vgit)
VERSION := $(shell echo $(VERSION) | sed 's/v//g')
COMMIT ?= $(shell git rev-parse --short HEAD || echo unknown)
BUILDTIME ?= $(shell date +%s)

UPDATER ?= off
LDFLAGS := -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.cssVersion=$(CSSVERSION) -X main.buildTimeUnix=$(BUILDTIME) $(if $(BUILTBY),-X 'main.builtBy=$(BUILTBY)',)
ifeq ($(UPDATER), on)
	LDFLAGS := $(LDFLAGS) -X main.updater=binary
else ifneq ($(UPDATER), off)
	LDFLAGS := $(LDFLAGS) -X main.updater=$(UPDATER)
endif



INTERNAL ?= on
TRAY ?= off
E2EE ?= off
TAGS := -tags "

ifeq ($(INTERNAL), on)
	DATA := data
else
	DATA := build/data
	TAGS := $(TAGS) external
endif

ifeq ($(TRAY), on)
	TAGS := $(TAGS) tray
endif

ifeq ($(E2EE), on)
	TAGS := $(TAGS) e2ee
endif

TAGS := $(TAGS)"

OS := $(shell go env GOOS)
ifeq ($(TRAY)$(OS), onwindows)
	LDFLAGS := $(LDFLAGS) -H=windowsgui
endif

DEBUG ?= off
ifeq ($(DEBUG), on)
	SOURCEMAP := --sourcemap
	TYPECHECK := npx tsc -noEmit --project ts/tsconfig.json
	# jank
	COPYTS := rm -r $(DATA)/web/js/ts; cp -r tempts $(DATA)/web/js/ts
	UNCSS := cp $(DATA)/web/css/bundle.css $(DATA)/bundle.css
	# TAILWIND := --content ""
else
	LDFLAGS := -s -w $(LDFLAGS)
	SOURCEMAP :=
	COPYTS :=
	TYPECHECK :=
	UNCSS := npx tailwindcss -i $(DATA)/web/css/bundle.css -o $(DATA)/bundle.css --content "html/crash.html"
	# UNCSS := npx uncss $(DATA)/crash.html --csspath web/css --output $(DATA)/bundle.css
	TAILWIND :=
endif

RACE ?= off
ifeq ($(RACE), on)
	RACEDETECTOR := -race
else
	RACEDETECTOR :=
endif

ifeq (, $(shell which esbuild))
	ESBUILDINSTALL := go install github.com/evanw/esbuild/cmd/esbuild@latest
else
	ESBUILDINSTALL :=
endif

ifeq ($(GOESBUILD), on)
	NPMIGNOREOPTIONAL := --no-optional
	NPMOPTS := $(NPMIGNOREOPTIONAL); $(ESBUILDINSTALL)
else
	NPMOPTS :=
endif

ifeq (, $(shell which swag))
	SWAGINSTALL := $(GOBINARY) install github.com/swaggo/swag/cmd/swag@latest
else
	SWAGINSTALL :=
endif

npm:
	$(info installing npm dependencies)
	npm install $(NPMOPTS)

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
	$(adding dark variants to typescript)
	rm -rf tempts
	cp -r ts tempts
	scripts/dark-variant.sh tempts
	scripts/dark-variant.sh tempts/modules
	$(info compiling typescript)
	mkdir -p $(DATA)/web/js
	$(ESBUILD) --target=es6 --bundle tempts/admin.ts $(SOURCEMAP) --outfile=./$(DATA)/web/js/admin.js --minify
	$(ESBUILD) --target=es6 --bundle tempts/user.ts $(SOURCEMAP) --outfile=./$(DATA)/web/js/user.js --minify
	$(ESBUILD) --target=es6 --bundle tempts/pwr.ts $(SOURCEMAP) --outfile=./$(DATA)/web/js/pwr.js --minify
	$(ESBUILD) --target=es6 --bundle tempts/pwr-pin.ts $(SOURCEMAP) --outfile=./$(DATA)/web/js/pwr-pin.js --minify
	$(ESBUILD) --target=es6 --bundle tempts/form.ts $(SOURCEMAP) --outfile=./$(DATA)/web/js/form.js --minify
	$(ESBUILD) --target=es6 --bundle tempts/setup.ts $(SOURCEMAP) --outfile=./$(DATA)/web/js/setup.js --minify
	$(ESBUILD) --target=es6 --bundle tempts/crash.ts --outfile=./$(DATA)/crash.js --minify
	$(COPYTS)

swagger:
	$(SWAGINSTALL)
	swag init -g main.go

compile:
	$(info Downloading deps)
	$(GOBINARY) mod download
	$(info Building)
	mkdir -p build
	$(GOBINARY) build $(RACEDETECTOR) -ldflags="$(LDFLAGS)" $(TAGS) -o build/jfa-go

compress:
	upx --lzma build/jfa-go

bundle-css:
	mkdir -p $(DATA)/web/css
	$(info copying fonts)
	cp -r node_modules/remixicon/fonts/remixicon.css node_modules/remixicon/fonts/remixicon.woff2 $(DATA)/web/css/
	$(info bundling css)
	$(ESBUILD) --bundle css/base.css --outfile=$(DATA)/web/css/bundle.css --external:remixicon.css --external:../fonts/hanken* --minify
	npx tailwindcss -i $(DATA)/web/css/bundle.css -o $(DATA)/web/css/bundle.css $(TAILWIND)
	# npx postcss -o $(DATA)/web/css/bundle.css $(DATA)/web/css/bundle.css

inline-css:
	cp html/crash.html $(DATA)/crash.html
	$(UNCSS)
	node scripts/inline.js root $(DATA) $(DATA)/crash.html $(DATA)/crash.html
	rm $(DATA)/bundle.css

variants-html:
	$(info copying html)
	cp -r html $(DATA)/
	$(info adding dark variants to html)
	node scripts/missing-colors.js html $(DATA)/html

copy:
	$(info copying crash page)
	mv $(DATA)/crash.html $(DATA)/html/
	$(info copying static data)
	mkdir -p $(DATA)/web
	cp -r static/* $(DATA)/web/
	$(info copying systemd service)
	cp jfa-go.service $(DATA)/
	$(info copying language files)
	cp -r lang $(DATA)/
	cp LICENSE $(DATA)/
	mv $(DATA)/web/css/bundle.css $(DATA)/web/css/$(CSSVERSION)bundle.css

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

quick: configuration typescript variants-html bundle-css inline-css copy compile

all: configuration npm email typescript variants-html bundle-css inline-css swagger copy compile
