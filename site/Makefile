.PHONY: clean
.DEFAULT_GOAL := all

GOESBUILD ?= off
ifeq ($(GOESBUILD), on)
	ESBUILD := esbuild
else
	ESBUILD := npx esbuild
endif

out/modal.css: ../css/modal.css
	mkdir -p out/
	cp ../css/modal.css out/

out/index.html: index.html
	mkdir -p out/
	NOTEMPLATE=1 node ../scripts/missing-colors.js index.html out/index.html
	node inject.js

out/remixicon.css: node_modules/remixicon/fonts/remixicon.css
	cp node_modules/remixicon/fonts/remixicon.css node_modules/remixicon/fonts/remixicon.woff2 out/

out/main.js: $(wildcard ts/**/*)
	-rm -r tempts
	cp -r ts tempts
	../scripts/dark-variant.sh tempts
	$(ESBUILD) --target=es6 --bundle tempts/main.ts --outfile=out/main.js --minify

out/bundle.css: out/modal.css out/remixicon.css out/main.js out/index.html
	-rm out/bundle.css
	$(ESBUILD) --bundle base.css --outfile=out/bundle.css --external:remixicon.css --external:modal.css --external:../fonts/hanken* --minify
	npx tailwindcss -c tailwind.config.js -i out/bundle.css -o out/bundle.css


out/favicon.ico: $(wildcard ../static/**/*)
	cp -r ../static/* out/

all: out/bundle.css out/favicon.ico

clean:
	-rm -rf out/
	-rm -rf tempts/
