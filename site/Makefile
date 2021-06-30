all:
	-mkdir -p out
	cp index.html ../css/modal.css out/
	cp -r node_modules/remixicon/fonts/remixicon.css node_modules/remixicon/fonts/remixicon.woff2 out/
	npx esbuild --bundle ts/main.ts --outfile=out/main.js --minify
	npx esbuild --bundle base.css --outfile=out/bundle.css --external:remixicon.css --external:modal.css --minify
	cd out && npx uncss index.html --stylesheets bundle.css > _bundle.css; cd ..
	mv out/_bundle.css out/bundle.css
	cd out && npx uncss index.html --stylesheets remixicon.css > _remixicon.css; cd ..
	mv out/_remixicon.css out/remixicon.css
	cp ../static/* out/

debug:
	-mkdir -p out
	cp index.html out/
	cp -r node_modules/remixicon/fonts/remixicon.css node_modules/remixicon/fonts/remixicon.woff2 out/
	npx esbuild --bundle base.css --outfile=out/bundle.css --external:remixicon.css --minify
	npx esbuild --bundle ts/main.ts --sourcemap --outfile=out/main.js --minify
	cp ../static/* out/

monitor:
	npx live-server --watch=out --open=out/index.html &
	npx nodemon -e html,css,ts -i out --exec "make debug"
