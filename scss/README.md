## SCSS

* `bs<4/5>-jf.scss` contains the source for the customizations to bootstrap. To customize the UI, you can make modifications to this file and then compile it.

**Note**: It is assumed that Bootstrap 5 is installed in `../../node_modules/bootstrap` relative to itself, and Bootstrap 4 in `../../node_modules/bootstrap4`.

* Compilation requires dev dependencies (`poetry update`), bootstrap and some extra npm packages.
* If you're buildings from source, you can simply run `poetry run task compile-css` before building to automatically get deps and compile CSS.
* If you are creating custom css, run `poetry run task get-npm-deps` to only install the necessary dependencies. Follow along with the commands `scss/compile.py` runs to build your css and then set `custom_css` in your config as the path to your minified css and change the `theme` option to `Custom CSS`.

