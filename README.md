# ![jfa-go](images/jfa-go-banner-wide.svg)

A rewrite of [jellyfin-accounts](https://github.com/hrfee/jellyfin-accounts) in Go. It has feature parity\* with the Python version, but should be faster.

\* *currently missing SMTP support.*

#### Install/usage

Grab an archive from the release section for your platform, and extract `jfa-go` and `data` to the same directory.
Run the executable to start.
```
Usage of ./jfa-go:
  -config string
    	alternate path to config file. (default "~/.config/jfa-go/config.ini")
  -data string
    	alternate path to data directory. (default "~/.config/jfa-go")
  -host string
    	alternate address to host web ui on.
  -port int
    	alternate port to host web ui on.
```

To switch from jf-accounts, copy your existing `~/.jf-accounts` to:

* `XDG_CONFIG_DIR/jfa-go` (usually ~/.config) on \*nix systems, 
* `%AppData%/jfa-go` on Windows,
* `~/Library/Application Support/jfa-go` on macOS.

(*or specify config/data path  with `-config/-data` respectively.*)

This is the first time i've even touched Go, and the code is a mess, so help is very welcome.
