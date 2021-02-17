GOESBUILD ?= off
ifeq ($(GOESBUILD), on)
	ESBUILD := esbuild
else
	ESBUILD := npx esbuild
endif
GOBINARY ?= go

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
	-$(ESBUILD) --bundle ts/form.ts --outfile=./data/web/js/form.js --minify
	-$(ESBUILD) --bundle ts/setup.ts --outfile=./data/web/js/setup.js --minify

ts-debug:
	$(info compiling typescript w/ sourcemaps)
	-mkdir -p data/web/js
	-$(ESBUILD) --bundle ts/admin.ts --sourcemap --outfile=./data/web/js/admin.js
	-$(ESBUILD) --bundle ts/form.ts --sourcemap --outfile=./data/web/js/form.js
	-$(ESBUILD) --bundle ts/setup.ts --sourcemap --outfile=./data/web/js/setup.js
	-rm -r data/web/js/ts
	$(info copying typescript)
	cp -r ts data/web/js

swagger:
	$(GOBINARY) get github.com/swaggo/swag/cmd/swag
	swag init -g main.go

version:
	python3 scripts/version.py auto

compile:
	$(info Downloading deps)
	$(GOBINARY) mod download
	$(info Building)
	mkdir -p build
	cd build && CGO_ENABLED=0 $(GOBINARY) build -ldflags="-s -w" -o ./jfa-go ../*.go

compile-debug:
	$(info Downloading deps)
	$(GOBINARY) mod download
	$(info Building)
	mkdir -p build
	cd build && CGO_ENABLED=0 $(GOBINARY) build -o ./jfa-go ../*.go

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

embed:
	python scripts/embed.py internal

noembed:
	python scripts/embed.py external
	-mkdir -p build
	$(info copying internal data into build/)
	cp -r data build/

install:
	cp -r build $(DESTDIR)/jfa-go

all: configuration npm email version typescript bundle-css swagger copy embed compile
all-external: configuration npm email version ts-debug bundle-css swagger copy noembed compile
debug: configuration npm email version ts-debug bundle-css swagger copy noembed compile-debug
