GOESBUILD ?= off
ifeq ($(GOESBUILD), on)
	ESBUILD := esbuild
else
	ESBUILD := npx esbuild
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
	-mkdir -p build/data
	python3 config/fixconfig.py -i config/config-base.json -o build/data/config-base.json
	$(info Generating config-default.ini)
	python3 config/generate_ini.py -i config/config-base.json -o build/data/config-default.ini

email:
	$(info Generating email html)
	python3 mail/generate.py -o build/data/

typescript:
	$(info compiling typescript)
	-mkdir -p build/data/web/js
	-$(ESBUILD) --bundle ts/admin.ts --outfile=./build/data/web/js/admin.js --minify
	-$(ESBUILD) --bundle ts/form.ts --outfile=./build/data/web/js/form.js --minify
	-$(ESBUILD) --bundle ts/setup.ts --outfile=./build/data/web/js/setup.js --minify

ts-debug:
	$(info compiling typescript w/ sourcemaps)
	-mkdir -p build/data/web/js
	-$(ESBUILD) --bundle ts/admin.ts --sourcemap --outfile=./build/data/web/js/admin.js
	-$(ESBUILD) --bundle ts/form.ts --sourcemap --outfile=./build/data/web/js/form.js
	-$(ESBUILD) --bundle ts/setup.ts --sourcemap --outfile=./build/data/web/js/setup.js
	-rm -r build/data/web/js/ts
	$(info copying typescript)
	cp -r ts build/data/web/js

swagger:
	go get github.com/swaggo/swag/cmd/swag
	swag init -g main.go

version:
	python3 version.py auto version.go

compile:
	$(info Downloading deps)
	go mod download
	$(info Building)
	mkdir -p build
	CGO_ENABLED=0 go build -o build/jfa-go *.go

compress:
	upx --lzma build/jfa-go

bundle-css:
	-mkdir -p build/data/web/css
	$(info bundling css)
	$(ESBUILD) esbuild --bundle css/base.css --outfile=build/data/web/css/bundle.css --external:remixicon.css --minify

copy:
	$(info copying fonts)
	cp -r node_modules/remixicon/fonts/remixicon.css node_modules/remixicon/fonts/remixicon.woff2 build/data/web/css/
	$(info copying html)
	cp -r html build/data/
	$(info copying static data)
	-mkdir -p build/data/web
	cp -r static/* build/data/web/
	$(info copying language files)
	cp -r lang build/data/


install:
	cp -r build $(DESTDIR)/jfa-go

all: configuration npm email version typescript bundle-css swagger compile copy
debug: configuration npm email version ts-debug bundle-css swagger compile copy
