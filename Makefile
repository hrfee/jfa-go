.PHONY: configuration email typescript swagger copy compile compress inline-css variants-html install clean npm config-description config-default precompile test
.DEFAULT_GOAL := all

GOESBUILD ?= off
ifeq ($(GOESBUILD), on)
	ESBUILD := esbuild
else
	ESBUILD := npx esbuild
endif
GOBINARY ?= go

CSSVERSION ?= $(shell git describe --tags --abbrev=0)
CSS_BUNDLE = $(DATA)/web/css/$(CSSVERSION)bundle.css

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
E2EE ?= on
TAGS := -tags "

ifeq ($(INTERNAL), on)
	DATA := build/data
	COMPDEPS := $(BUILDDEPS)
else
	DATA := build/data
	TAGS := $(TAGS) external
	COMPDEPS :=
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
	MINIFY := 
	TYPECHECK := npx tsc -noEmit --project ts/tsconfig.json
	# jank
	COPYTS := rm -r $(DATA)/web/js/ts; cp -r tempts $(DATA)/web/js/ts
	UNCSS := cp $(CSS_BUNDLE) $(DATA)/bundle.css
	# TAILWIND := --content ""
else
	LDFLAGS := -s -w $(LDFLAGS)
	SOURCEMAP :=
	MINIFY := --minify
	COPYTS :=
	TYPECHECK :=
	UNCSS := npx tailwindcss -i $(CSS_BUNDLE) -o $(DATA)/bundle.css --content "html/crash.html"
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
	SWAGINSTALL := $(GOBINARY) install github.com/swaggo/swag/cmd/swag@v1.16.4
else
	SWAGINSTALL :=
endif

# FLAG HASHING: To rebuild on flag change.
# credit for idea to https://bnikolic.co.uk/blog/sh/make/unix/2021/07/08/makefile.html
rebuildFlags := GOESBUILD GOBINARY VERSION COMMIT UPDATER INTERNAL TRAY E2EE TAGS DEBUG RACE
rebuildVals := $(foreach v,$(rebuildFlags),$(v)=$($(v)))
rebuildHash := $(strip $(shell echo $(rebuildVals) | sha256sum | cut -d " " -f1))
rebuildHashFile := $(DATA)/buildhash-$(rebuildHash).txt

CONFIG_BASE = config/config-base.yaml

# CONFIG_DESCRIPTION = $(DATA)/config-base.json
CONFIG_DEFAULT = $(DATA)/config-default.ini
# $(CONFIG_DESCRIPTION) &: $(CONFIG_BASE)
# 	$(info Fixing config-base)
# 	-mkdir -p $(DATA)

$(DATA):
	mkdir -p $(DATA)/web/js
	mkdir -p $(DATA)/web/css

$(CONFIG_DEFAULT): $(CONFIG_BASE)
	$(info Generating config-default.ini)
	go run scripts/ini/main.go -in $(CONFIG_BASE) -out $(DATA)/config-default.ini

configuration: $(CONFIG_DEFAULT)

EMAIL_SRC = $(wildcard mail/*)
EMAIL_TARGET = $(DATA)/confirmation.html
$(EMAIL_TARGET): $(EMAIL_SRC)
	$(info Generating email html)
	npx mjml mail/*.mjml -o $(DATA)/
	$(info Copying plaintext mail)
	cp mail/*.txt $(DATA)/

TYPESCRIPT_FULLSRC = $(shell find ts/ -type f -name "*.ts")
TYPESCRIPT_SRC = $(wildcard ts/*.ts)
TYPESCRIPT_TEMPSRC = $(TYPESCRIPT_SRC:ts/%=tempts/%)
# TYPESCRIPT_TARGET = $(patsubst %.ts,%.js,$(subst tempts/,./$(DATA)/web/js/,$(TYPESCRIPT_TEMPSRC)))
TYPESCRIPT_TARGET = $(DATA)/web/js/admin.js
$(TYPESCRIPT_TARGET): $(TYPESCRIPT_FULLSRC) ts/tsconfig.json
	$(TYPECHECK)
	# rm -rf tempts
	# cp -r ts tempts
	rm -rf tempts
	mkdir -p tempts
	$(adding dark variants to typescript)
	# scripts/dark-variant.sh tempts
	# scripts/dark-variant.sh tempts/modules
	go run scripts/variants/main.go -dir ts -out tempts 
	$(info compiling typescript)
	$(foreach tempsrc,$(TYPESCRIPT_TEMPSRC),$(ESBUILD) --target=es6 --bundle $(tempsrc) $(SOURCEMAP) --outfile=$(patsubst %.ts,%.js,$(subst tempts/,./$(DATA)/web/js/,$(tempsrc))) $(MINIFY);)
	$(COPYTS)

SWAGGER_SRC = $(wildcard api*.go) $(wildcard *auth.go) views.go
SWAGGER_TARGET = docs/docs.go
$(SWAGGER_TARGET): $(SWAGGER_SRC)
	$(SWAGINSTALL)
	swag init --parseDependency --parseInternal -g main.go

VARIANTS_SRC = $(wildcard html/*.html)
VARIANTS_TARGET = $(DATA)/html/admin.html
$(VARIANTS_TARGET): $(VARIANTS_SRC)
	$(info copying html)
	cp -r html $(DATA)/
	$(info adding dark variants to html)
	node scripts/missing-colors.js html $(DATA)/html

ICON_SRC = node_modules/remixicon/fonts/remixicon.css node_modules/remixicon/fonts/remixicon.woff2
ICON_TARGET = $(ICON_SRC:node_modules/remixicon/fonts/%=$(DATA)/web/css/%)
SYNTAX_LIGHT_SRC = node_modules/highlight.js/styles/base16/atelier-sulphurpool-light.min.css
SYNTAX_LIGHT_TARGET = $(DATA)/web/css/$(CSSVERSION)highlightjs-light.css
SYNTAX_DARK_SRC = node_modules/highlight.js/styles/base16/circus.min.css
SYNTAX_DARK_TARGET = $(DATA)/web/css/$(CSSVERSION)highlightjs-dark.css
CODEINPUT_SRC = node_modules/@webcoder49/code-input/code-input.min.css
CODEINPUT_TARGET = $(DATA)/web/css/$(CSSVERSION)code-input.css
CSS_SRC = $(wildcard css/*.css)
CSS_TARGET = $(DATA)/web/css/part-bundle.css
CSS_FULLTARGET = $(CSS_BUNDLE)
ALL_CSS_SRC = $(ICON_SRC) $(CSS_SRC) $(SYNTAX_LIGHT_SRC) $(SYNTAX_DARK_SRC)
ALL_CSS_TARGET = $(ICON_TARGET)

$(CSS_FULLTARGET): $(TYPESCRIPT_TARGET) $(VARIANTS_TARGET) $(ALL_CSS_SRC) $(wildcard html/*.html)
	$(info copying fonts)
	cp -r node_modules/remixicon/fonts/remixicon.css node_modules/remixicon/fonts/remixicon.woff2 $(DATA)/web/css/
	cp -r $(SYNTAX_LIGHT_SRC) $(SYNTAX_LIGHT_TARGET)
	cp -r $(SYNTAX_DARK_SRC) $(SYNTAX_DARK_TARGET)
	cp -r $(CODEINPUT_SRC) $(CODEINPUT_TARGET)
	$(info bundling css)
	rm -f $(CSS_TARGET) $(CSS_FULLTARGET)
	$(ESBUILD) --bundle css/base.css --outfile=$(CSS_TARGET) --external:remixicon.css --external:../fonts/hanken* --minify

	npx tailwindcss -i $(CSS_TARGET) -o $(CSS_FULLTARGET) $(TAILWIND)
	rm $(CSS_TARGET)
	# mv $(CSS_BUNDLE) $(DATA)/web/css/$(CSSVERSION)bundle.css
	# npx postcss -o $(CSS_TARGET) $(CSS_TARGET)

INLINE_SRC = html/crash.html
INLINE_TARGET = $(DATA)/crash.html
$(INLINE_TARGET): $(CSS_FULLTARGET) $(INLINE_SRC)
	cp html/crash.html $(DATA)/crash.html
	$(UNCSS) # generates $(DATA)/bundle.css for us
	node scripts/inline.js root $(DATA) $(DATA)/crash.html $(DATA)/crash.html
	rm $(DATA)/bundle.css

LANG_SRC = $(shell find ./lang)
LANG_TARGET = $(LANG_SRC:lang/%=$(DATA)/lang/%)
STATIC_SRC = $(wildcard static/*)
STATIC_TARGET = $(STATIC_SRC:static/%=$(DATA)/web/%)
COPY_SRC = images/banner.svg jfa-go.service LICENSE $(LANG_SRC) $(STATIC_SRC)
COPY_TARGET = $(DATA)/jfa-go.service
# $(DATA)/LICENSE $(LANG_TARGET) $(STATIC_TARGET) $(DATA)/web/css/$(CSSVERSION)bundle.css
$(COPY_TARGET): $(INLINE_TARGET) $(STATIC_SRC) $(LANG_SRC) $(CONFIG_BASE)
	$(info copying $(CONFIG_BASE))
	go run scripts/yaml/main.go -in $(CONFIG_BASE) -out $(DATA)/$(shell basename $(CONFIG_BASE))
	$(info copying crash page)
	cp $(DATA)/crash.html $(DATA)/html/
	$(info copying static data)
	cp images/banner.svg static/banner.svg
	cp -r static/* $(DATA)/web/
	$(info copying systemd service)
	cp jfa-go.service $(DATA)/
	$(info copying language files)
	cp -r lang $(DATA)/
	cp LICENSE $(DATA)/

BUILDDEPS := $(DATA) $(CONFIG_DEFAULT) $(EMAIL_TARGET) $(COPY_TARGET) $(SWAGGER_TARGET) $(INLINE_TARGET) $(CSS_FULLTARGET) $(TYPESCRIPT_TARGET) 
precompile: $(BUILDDEPS)

COMPDEPS = $(rebuildHashFile)
ifeq ($(INTERNAL), on)
	COMPDEPS = $(BUILDDEPS) $(rebuildHashFile)
endif

$(rebuildHashFile):
	$(info recording new flags $(rebuildVals))
	rm -f $(DATA)/buildhash-*.txt
	touch $(rebuildHashFile)

GO_SRC = $(shell find ./ -name "*.go")
GO_TARGET = build/jfa-go
$(GO_TARGET): $(COMPDEPS) $(SWAGGER_TARGET) $(GO_SRC) go.mod go.sum
	$(info Downloading deps)
	$(GOBINARY) mod download
	$(info Building)
	mkdir -p build
	$(GOBINARY) build $(RACEDETECTOR) -ldflags="$(LDFLAGS)" $(TAGS) -o $(GO_TARGET) 

test: $(BUILDDEPS) $(COMPDEPS) $(SWAGGER_TARGET) $(GO_SRC) go.mod go.sum
	$(GOBINARY) test -ldflags="$(LDFLAGS)" $(TAGS) -p 1

all: $(BUILDDEPS) $(GO_TARGET) $(rebuildHashFile)

compress:
	upx --lzma $(GO_TARGET)

install:
	cp -r build $(DESTDIR)/jfa-go

clean:
	-rm -r $(DATA)
	-rm -r build
	-rm mail/*.html
	-rm docs/docs.go docs/swagger.json docs/swagger.yaml
	go clean

npm:
	$(info installing npm dependencies)
	npm install $(NPMOPTS)
