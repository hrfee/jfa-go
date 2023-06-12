all:
	-mkdir -p out
	cp ../css/modal.css out/
	NOTEMPLATE=1 node ../scripts/missing-colors.js index.html out/index.html
	cp -r node_modules/remixicon/fonts/remixicon.css node_modules/remixicon/fonts/remixicon.woff2 out/
	-rm -r tempts
	cp -r ts tempts
	../scripts/dark-variant.sh tempts
	npx esbuild --target=es6 --bundle tempts/main.ts --outfile=out/main.js --minify
	npx esbuild --bundle base.css --outfile=out/bundle.css --external:remixicon.css --external:modal.css --minify
	npx tailwindcss -c tailwind.config.js -i out/bundle.css -o out/bundle.css
	cd out && npx uncss index.html --stylesheets remixicon.css > _remixicon.css; cd ..
	mv out/_remixicon.css out/remixicon.css
	cp ../static/* out/
	node inject.js

debug:
	-mkdir -p out
	cp ../css/modal.css out/
	NOTEMPLATE=1 node ../scripts/missing-colors.js index.html out/index.html
	cp -r node_modules/remixicon/fonts/remixicon.css node_modules/remixicon/fonts/remixicon.woff2 out/
	-rm -r tempts
	cp -r ts tempts
	../scripts/dark-variant.sh tempts
	npx esbuild --bundle base.css --outfile=out/bundle.css --external:remixicon.css --minify
	npx esbuild --target=es6 --bundle ts/main.ts --sourcemap --outfile=out/main.js --minify
	npx tailwindcss -c tailwind.config.js -i out/bundle.css -o out/bundle.css
	cp ../static/* out/

monitor:
	npx live-server --watch=out --open=out/index.html &
	npx nodemon -e html,css,ts -i out --exec "make debug"
