# This is an example PKGBUILD file. Use this as a start to creating your own,
# and remove these comments. For more information, see 'man PKGBUILD'.
# NOTE: Please fill out the license field for your package! If it is unknown,
# then please put 'unknown'.

# Maintainer: Harvey Tindall <hrfee@protonmail.ch>
pkgname="jfa-go"
pkgver=0.1.3
pkgrel=1
epoch=
pkgdesc="A web app for managing users on Jellyfin"
arch=("x86_64")
url="https://github.com/hrfee/jfa-go"
license=('MIT')
groups=()
depends=()
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
#source=("$pkgname-$pkgver.tar.gz"
#        "$pkgname-$pkgver.patch")
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
    make install DESTDIR="$pkgdir"/opt
    cp -r "$pkgdir"/opt/$pkgname /opt/$pkgname
    ln -s /opt/$pkgname/$pkgname /usr/bin/$pkgname
}

# check() {
# 	cd "$pkgname-$pkgver"
# 	make -k check
# }
# 
# package() {
# 	cd "$pkgname-$pkgver"
# 	make DESTDIR="$pkgdir/" install
# }
