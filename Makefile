npm:
	$(info installing npm dependencies)
	npm install

configuration:
	$(info Fixing config-base)
	-mkdir -p build/data
	python3 config/fixconfig.py -i config/config-base.json -o build/data/config-base.json
	$(info Generating config-default.ini)
	python3 config/generate_ini.py -i config/config-base.json -o build/data/config-default.ini

email:
	$(info Generating email html)
	python3 mail/generate.py -o build/data/

ts:
	$(info compiling typescript)
	-mkdir -p build/data/web/js
	-npx esbuild ts/*.ts ts/modules/*.ts --outdir=./build/data/web/js/

ts-debug:
	$(info compiling typescript w/ sourcemaps)
	-mkdir -p build/data/web/js
	-npx esbuild ts/*.ts ts/modules/*.ts --sourcemap --outdir=./build/data/web/js/
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

copy:
	$(info copying css)
	-mkdir -p build/data/web/css
	cp -r css build/data/web/
	cp node_modules/a17t/dist/a17t.css build/data/web/css/
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

all: configuration npm email version ts swagger compile copy
debug: configuration npm email version ts-debug swagger compile copy
