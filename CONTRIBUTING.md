#### Code
I use 4 spaces for indentation. Go should ideally be formatted with `goimports` and/or `gofmt`. I don't use a formatter on typescript, so don't worry about that.

If you need to test your changes:
* `make debug` will build everything, and include sourcemaps for typescript. This should be the first thing you run.
* `make compile` compiles go into `build/jfa-go`.
* `make ts-debug` will compile typescript w/ sourcemaps into `build/data/web/js`.
* `make copy` will copy css, html, language and static files into `build/data`.
