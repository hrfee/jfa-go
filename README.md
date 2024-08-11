![jfa-go](images/banner.svg)
[![Build Status](https://ci.hrfee.dev/api/badges/3/status.svg)](https://ci.hrfee.dev/repos/3)
[![Docker Hub](https://img.shields.io/docker/pulls/hrfee/jfa-go?label=docker)](https://hub.docker.com/r/hrfee/jfa-go)
[![Translation status](https://weblate.jfa-go.com/widgets/jfa-go/-/svg-badge.svg)](https://weblate.jfa-go.com/engage/jfa-go/)
[![Docs/Wiki](https://img.shields.io/static/v1?label=documentation&message=jfa-go.com&color=informational)](https://wiki.jfa-go.com)
[![Discord](https://img.shields.io/discord/922842034170122321?color=%235865F2&label=discord)](https://discord.com/invite/MrtvuQmyhP)

##### Downloads:
##### [docker](#docker) | [debian/ubuntu](#debian) | [arch (aur)](#aur) | [other platforms](#other-platforms)

---
##  Project Status: Active-ish
Studies mean I can't work on this project a lot outside of breaks, however I hope i'll be able to fit in general support and things like bug fixes into my time. New features and such will likely come in short bursts throughout the year (if they do at all).

#### Does/Will it still work?
jfa-go currently works on Jellyfin 10.9.8, the latest version as of 31/07/2024. I should be able to maintain compatability in the future, unless any big changes occur.

#### Alternatives
If you want a bit more of a guarantee of support, I've seen these projects mentioned although haven't tried them myself.

* [Wizarr](https://github.com/Wizarrrr/wizarr) focuses on invites, and also includes some Discord & Ombi integration.
* [Jellyseerr](https://github.com/Fallenbagel/jellyseerr) is a fork of Overseerr which can manage users and mainly acts as an Ombi alternative.
  * [jfa-go now integrates with Jellyseerr, much like Ombi, but better.](https://github.com/hrfee/jfa-go/pull/351)
* [Organizr](https://github.com/causefx/Organizr) doesn't focus on Jellyfin, but allows putting self-hosted services into "tabs" on a central page, and allows creating users, which lets one control who can access what.
---
jfa-go is a user management app for [Jellyfin](https://github.com/jellyfin/jellyfin) (and [Emby](https://emby.media/) as 2nd class) that provides invite-based account creation as well as other features that make one's instance much easier to manage.

#### Features
* 🧑 Invite based account creation: Send invites to your friends or family, and let them choose their own username and password without relying on you.
    * Send invites via a link and/or email, discord, telegram or matrix
    * Granular control over invites: Validity period as well as number of uses can be specified.
    * Account profiles: Assign settings profiles to invites so new users have your predefined permissions, homescreen layout, etc. applied to their account on creation.
    * Password validation: Ensure users choose a strong password.
    * CAPTCHAs and contact method verificatoin can be enabled to avoid bots.
* ⌛ User expiry: Specify a validity period, and new users accounts will be disabled/deleted after it. The period can be manually extended too.
* 🔗 Ombi/Jellyseerr Integration: Automatically creates and synchronizes details for new accounts. Supports setting permissions with the Profiles feature. **Ombi integration use is risky, see [wiki](https://wiki.jfa-go.com/docs/ombi/)**.
* Account management: Bulk or individually; apply settings, delete, disable/enable, send messages and much more.
* 📣 Announcements: Bulk message your users with announcements about your server.
* Telegram/Discord/Matrix Integration: Verify users via a chat bot, and send Password Resets, Announcements, etc. through it.
* "My Account" Page: Allows users to reset their password, manage contact details, view their account expiry date, and send referrals. Can be customized with markdown.
* Referrals: Users can be given special invites to send to their friends and families, similar to some invite-only services like Bluesky.
* 🔑 Password resets: When users forget their passwords and request a change in Jellyfin, jfa-go reads the PIN from the created file and sends it straight to them via email/telegram.
  * Can also be done through the "My Account" page if enabled.
* Admin Notifications: Get notified when someone creates an account, or an invite expires.
* 🌓 Customizations
    * Customize emails with variables and markdown
    * Specify contact and help messages to appear in emails and pages
    * Light and dark themes available

#### Interface
<p align="center">
    <img src="images/invites.png" width="47%" style="margin-left: 1.5%;" align="top" alt="Invites tab"></img>
    <img src="images/create.png" width="47%" style="margin-right: 1.5%;" align="top" alt="Accounts creation"></img> 
    <img src="images/myaccount.png" width="47%" style="margin-left: 1.5%; margin-top: 1rem;" align="top" alt="My Account Page"></img>
    <img src="images/accounts.png" width="47%" style="margin-right: 1.5%; margin-top: 1rem;" align="top" alt="Accounts tab"></img> 
</p>

#### Install

**Note**: `TrayIcon` builds include a tray icon to start/stop/restart, and an option to automatically start when you log-in to your computer. For Linux users, these builds depend on the `libappindicator3-1`/`libappindicator-gtk3`/`libappindicator` package for Debian/Ubuntu, Fedora, and Alpine respectively.

`MatrixE2EE` builds (and Linux `TrayIcon` builds) include support for end-to-end encryption for the Matrix bot, but require the `libolm(-dev)` dependency. `.deb/.rpm/.apk` packages list this dependency, and docker images include it.

##### [Docker](https://hub.docker.com/r/hrfee/jfa-go)
```sh
docker create \
             --name "jfa-go" \ # Whatever you want to name it
             -p 8056:8056 \
            # -p 8057:8057 if using tls
             -v /path/to/.config/jfa-go:/data \ # Path to wherever you want to store the config file and other data
             -v /path/to/jellyfin:/jf \ # Only needed for password resets through Jellyfin, ignore if not using or using Emby
             -v /etc/localtime:/etc/localtime:ro \ # Makes sure time is correct
             hrfee/jfa-go # hrfee/jfa-go:unstable for latest build from git
```

##### [Debian/Ubuntu](https://apt.hrfee.dev)
```sh
sudo apt-get update && sudo apt-get install curl apt-transport-https gnupg
curl https://apt.hrfee.dev/hrfee.pubkey.gpg | gpg --dearmor | sudo tee /etc/apt/trusted.gpg.d/apt.hrfee.dev.gpg

# For stable releases
echo "deb https://apt.hrfee.dev trusty main" | sudo tee /etc/apt/sources.list.d/hrfee.list
# ------
# For unstable releases
echo "deb https://apt.hrfee.dev trusty-unstable main" | sudo tee /etc/apt/sources.list.d/hrfee.list
# ------

sudo apt-get update

# For servers
sudo apt-get install jfa-go
# ------
# For desktops/servers with GUI (may pull in lots of dependencies)
sudo apt-get install jfa-go-tray
# ------
```

##### Arch
Available on the AUR as:
* [jfa-go](https://aur.archlinux.org/packages/jfa-go/) (stable)
* [jfa-go-bin](https://aur.archlinux.org/packages/jfa-go) (pre-compiled, stable)
* [jfa-go-git](https://aur.archlinux.org/packages/jfa-go-git/) (nightly)

##### Other platforms
Download precompiled binaries from:
 * [The releases section](https://github.com/hrfee/jfa-go/releases) (stable)
 * [dl.jfa-go.com](https://dl.jfa-go.com) (nightly)

unzip the `jfa-go`/`jfa-go.exe` executable to somewhere useful.
* For \*nix/macOS users, `chmod +x jfa-go` then place it somewhere in your PATH like `/usr/bin`.

Run the executable to start.


#### Build from source
If you're using docker, a Dockerfile is provided that builds from source.

Otherwise, full build instructions can be found [here](https://wiki.jfa-go.com/docs/build/).

#### Usage
Simply run `jfa-go` to start the application. A setup wizard will start on `localhost:8056` (or your own specified address). Upon completion, refresh the page.

```
Usage of jfa-go:
  start
	start jfa-go as a daemon and run in the background.
  stop
	stop a daemonized instance of jfa-go.
  systemd
	generate a systemd .service file.

  -config, -c string
    	alternate path to config file. (default "/home/hrfee/.config/jfa-go/config.ini")
  -data, -d string
    	alternate path to data directory. (default "/home/hrfee/.config/jfa-go")
  -debug
    	Enables debug logging.
  -help, -h
    	prints this message.
  -host string
    	alternate address to host web ui on.
  -port, -p int
    	alternate port to host web ui on.
  -pprof
    	Exposes pprof profiler on /debug/pprof.
  -restore string
    	path to database backup to restore.
  -swagger
    	Enable swagger at /swagger/index.html
```

#### Systemd
jfa-go does not run as a daemon by default. Run `jfa-go systemd` to create a systemd `.service` file in your current directory, which you can copy into `~/.config/systemd/user` or somewhere else.


#### Contributing
See [the wiki page](https://wiki.jfa-go.com/docs/dev/).
##### Translation
[![Translation status](https://weblate.jfa-go.com/widgets/jfa-go/-/multi-auto.svg)](https://weblate.jfa-go.com/engage/jfa-go/)

For translations, use the weblate instance [here](https://weblate.jfa-go.com/engage/jfa-go/). You can login with github.

#### Sponsors
Big thanks to those who sponsor me. You can see them below:

[<img src="https://sponsors-endpoint.hrfee.pw/sponsor/avatar/0" width="35">](https://sponsors-endpoint.hrfee.pw/sponsor/profile/0)
