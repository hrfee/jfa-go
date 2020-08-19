# Maintainer: Harvey Tindall <hrfee@protonmail.ch>
pkgname="jfa-go"
pkgver=0.1.3
pkgrel=1
pkgdesc="A web app for managing users on Jellyfin"
arch=("x86_64")
url="https://github.com/hrfee/jfa-go"
license=('MIT')
makedepends=('go>=1.14' 'python>=3.6.0-1' 'nodejs' 'npm')
checkdepends=()
optdepends=()
provides=()
conflicts=()
replaces=()
backup=()
options=()
install=
changelog=
source=("jfa-go::git+https://github.com/hrfee/jfa-go.git")
noextract=()
md5sums=(SKIP)
validpgpkeys=()

prepare() {
	cd jfa-go
    ls
    make configuration sass-headless mail-headless
}

build() {
	cd jfa-go
	make compile
}

package() {
    cd jfa-go
    make copy
    install -d "$pkgdir"/opt
    make install DESTDIR="$pkgdir"/opt
}
