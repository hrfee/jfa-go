#### Translation
Currently only the account creation form can be translated. Strings are defined in `lang/form/<country-code>.json` (country code as in `en-us`, `fr-fr`, e.g). You can see the existing ones [here](https://github.com/hrfee/jfa-go/tree/main/lang/form).
Make sure to define `name` in the `meta` section, and you can optionally add an `author` value there as well. If you can, make a pull request with your new file. If not, email me or create an issue.

#### Code
I use 4 spaces for indentation. Go should ideally be formatted with `goimports` and/or `gofmt`. I don't use a formatter on typescript, so don't worry about that.

If you need to test your changes:
* `make debug` will build everything, and include sourcemaps for typescript. This should be the first thing you run.
* `make compile` compiles go into `build/jfa-go`.
* `make ts-debug` will compile typescript w/ sourcemaps into `build/data/web/js`.
* `make copy` will copy css, html, language and static files into `build/data`.
