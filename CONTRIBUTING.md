#### Code
I use 4 spaces for indentation. Go should ideally be formatted with `goimports` and/or `gofmt`. I don't use a formatter on typescript, so don't worry about that.

Code in Go should ideally use `PascalCase` for exported values, and `camelCase` for non-exported, JSON for transferring data should use `snake_case`, and Typescript should use `camelCase`. Forgive me for my many inconsistencies in this, and feel free to fix them if you want.

Functions in Go that need to access `*appContext` should be generally be receivers, except when the behaviour could be seen as somewhat independent from it (`email.go` is the best example, its behaviour is broadly independent from the main app except from a couple config values).


#### Compiling

Prefix each of these with `make DEBUG=on INTERNAL=off `:
* `all` will download deps and build everything. The executable and data will be placed in `build`. This is only necessary the first time.
* `compile` will only compile go code into the `build/jfa-go` executable.
* `typescript` will compile typescript w/ sourcemaps into `build/data/web/js`.
* `bundle-css` will bundle CSS and place it in `build/data/web/css`.
* `configuration` will generate the `config-base.json` (used to render settings in the web ui) and `config-default.ini` and put them in `build/data`.
* `email` will compile email mjml, and copy the text versions in to `build/data`.
* `copy` will copy iconography, html, language files and static data into `build/data`.

See the [wiki](https://github.com/hrfee/jfa-go/wiki/Build) for more info.
