configuration:
	$(info Fixing config-base)
	python3 config/fixconfig.py -i config/config-base.json -o data/config-base.json
	$(info Generating config-default.ini)
	python3 config/generate_ini.py -i config/config-base.json -o data/config-default.ini --version git

sass:
	$(info Getting libsass)
	python3 -m pip install libsass
	$(info Getting node dependencies)
	python3 scss/get_node_deps.py
	$(info Compiling sass)
	python3 scss/compile.py

sass-headless:
	$(info Getting libsass)
	python3 -m pip install libsass
	$(info Getting node dependencies)
	python3 scss/get_node_deps.py
	$(info Compiling sass)
	python3 scss/compile.py -y

mail-headless:
	$(info Generating email html)
	python3 mail/generate.py -y

mail:
	$(info Generating email html)
	python3 mail/generate.py

typescript:
	$(info Compiling typescript)
	-npx tsc -p ts/

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

all: configuration sass mail version typescript compile copy
headless: configuration sass-headless mail-headless version typescript compile copy



