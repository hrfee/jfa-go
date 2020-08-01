# jfa-go

A rewrite of [jellyfin-accounts](https://github.com/hrfee/jellyfin-accounts) in Go. Should be fully functional, and functions the same as jf-accounts. To switch, copy your existing `~/.jf-accounts` to:

* `XDG_CONFIG_DIR/jfa-go` (usually ~/.config) on \*nix systems, 
* `%AppData%/jfa-go` on Windows,
* `~/Library/Application Support/jfa-go` on macOS.

(*or specify config/data path  with `-config/-data` respectively.*)

Suggestions and help welcome.
