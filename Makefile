configuration:
	$(info Fixing config-base)
	python3 config/fixconfig.py -i config/config-base.json -o data/config-base.json
	$(info Generating config-default.ini)
	python3 config/generate_ini.py -i config/config-base.json -o data/config-default.ini

sass:
	$(info Getting libsass)
	python3 -m pip install libsass
	$(info Getting node dependencies)
	npm install
	$(info Compiling sass)
	python3 scss/compile.py

email:
	$(info Generating email html)
	python3 mail/generate.py

typescript:
	$(info Compiling typescript)
	npx esbuild ts/*.ts ts/modules/*.ts --outdir=data/static --minify
	-rm -r data/static/ts
	-rm -r data/static/typings
	-rm data/static/*.map

ts-debug:
	-npx tsc -p ts/ --sourceMap
	-rm -r data/static/ts
	-rm -r data/static/typings
	cp -r ts data/static/

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
	$(info Copying data)
	cp -r data build/

install:
	cp -r build $(DESTDIR)/jfa-go

all: configuration sass email version typescript swagger compile copy
debug: configuration sass email version ts-debug swagger compile copy
