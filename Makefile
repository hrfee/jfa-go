configuration:
	echo "Fixing config-base"
	python3 config/fixconfig.py -i config/config-base.json -o data/config-base.json
	echo "Generating config-default.ini"
	python3 config/generate_ini.py -i config/config-base.json -o data/config-default.ini

sass:
	echo "Getting libsass"
	python3 -m pip install libsass
	echo "Getting node dependencies"
	python3 scss/get_node_deps.py
	echo "Compiling sass"
	python3 scss/compile.py

sass-headless:
	echo "Getting libsass"
	python3 -m pip install libsass
	echo "Getting node dependencies"
	python3 scss/get_node_deps.py
	echo "Compiling sass"
	python3 scss/compile.py -y

mail-headless:
	echo "Generating email html"
	python3 mail/generate.py -y

mail:
	echo "Generating email html"
	python3 mail/generate.py

compile:
	echo "Downloading deps"
	go mod download
	echo "Building"
	mkdir -p build
	go build -o build/jfa-go *.go

copy:
	echo "Copying data"
	cp -r data build/

install:
	cp -r build $(DESTDIR)/jfa-go

all: configuration sass mail compile copy
headless: configuration sass-headless mail-headless copy



