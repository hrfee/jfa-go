---
title: "Building/Contributing for developers"
date: 2021-07-25T00:33:36+01:00
draft: false
---
# Code
I use 4 spaces for indentation. Go should ideally be formatted with `goimports` and/or `gofmt`. I don't use a formatter on typescript, so don't worry about that.

Code in Go should ideally use `PascalCase` for exported values, and `camelCase` for non-exported, JSON for transferring data should use `snake_case`, and Typescript should use `camelCase`. Forgive me for my many inconsistencies in this, and feel free to fix them if you want.

Functions in Go that need to access `*appContext` should be generally be receivers, except when the behaviour could be seen as somewhat independent from it (`email.go` is the best example, its behaviour is broadly independent from the main app except from a couple config values).


# Compiling

The Makefile is more suited towards development than other build methods, and provides separate build stages to speed up compilation when only making changes to specific aspects of the project.

Prefix each of these with `make DEBUG=on `:
* `all` will download deps and build everything. The executable and data will be placed in `build`. This is only necessary the first time.
* `npm` will download all node.js build-time dependencies.
* `compile` will only compile go code into the `build/jfa-go` executable.
* `typescript` will compile typescript w/ sourcemaps into `build/data/web/js`.
* `bundle-css` will bundle CSS and place it in `build/data/web/css`.
  * `inline` will inline the css and javascript used in the single-file crash report webpage.
* `configuration` will generate the `config-base.json` (used to render settings in the web ui) and `config-default.ini` and put them in `build/data`.
* `email` will compile email mjml, and copy the text versions in to `build/data`.
* `swagger`: generates swagger documentation for the API.
* `copy` will copy iconography, html, language files and static data into `build/data`.

## Environment variables

* `DEBUG=on/off`: If on, compiles with type-checking for typescript, sourcemaps, non-minified css and no symbol stripping.
* `INTERNAL=on/off`: Whether or not to embed file assets into the binary itself, or store them separately beside the binary.
* `UPDATER=on/off/docker`: Enable/Disable the updater, or set a special update type (currently only docker, which disables self-updating the binary).
* `TRAY=on/off`: Enable/disable the tray icon, which lets you start/stop/autostart on login. For linux, requires `libappindicator3-dev` for debian or the equivalent on other distributions.
* `GOESBUILD=on`: Use a locally installed `esbuild` binary. NPM doesn't provide builds for all os/architectures, so `npx esbuild` might not work for you, so the binary is compiled/installed with `go get`.
* `GOBINARY=<path to go>`: Alternative path to go executable. Useful for testing with unstable go releases.
* `VERSION=v<semver>`: Alternative verision number, useful to test update functionality.
* `COMMIT=<short commit>`: Self explanatory.
* `LDFLAGS=<ldflags>`: Passed to `go build -ldflags`.
* `E2EE=on/off`: Enable/disable end-to-end encryption support for Matrix, which is currently very broken. Must subsequently be enabled (with Advanced settings enabled) in Settings > Matrix.
* `TAGS=<tags>`: Passed to `go build -tags`.
* `OS=<os>`: Unrelated to GOOS, if set to `windows`, `-H=windowsgui` is passed to ldflags, which stops a windows terminal popping up when run.
* `RACE=on/off`: If on, compiles with the go race detector included.
