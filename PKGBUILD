# Maintainer: Your Name <your.email@example.com>
pkgname=mkanban-tui
pkgver=0.1.0
pkgrel=1
pkgdesc="A powerful terminal-based Kanban board system with git workflow integration"
arch=('x86_64' 'aarch64' 'armv7h')
url="https://github.com/blendonl/mkanban-tui"
license=('MIT')
depends=('glibc')
makedepends=('go' 'git')
optdepends=(
    'git: for git workflow integration'
    'tmux: for tmux session integration'
)
provides=('mkanban' 'mkanbad')
conflicts=('mkanban' 'mkanbad')
backup=('etc/mkanban/config.yaml')
source=("git+${url}.git#tag=v${pkgver}")
sha256sums=('SKIP')

# For local builds, use this instead:
# source=("${pkgname}::git+file://$(pwd)")

build() {
    cd "${srcdir}/${pkgname}"

    export CGO_CPPFLAGS="${CPPFLAGS}"
    export CGO_CFLAGS="${CFLAGS}"
    export CGO_CXXFLAGS="${CXXFLAGS}"
    export CGO_LDFLAGS="${LDFLAGS}"
    export GOFLAGS="-buildmode=pie -trimpath -ldflags=-linkmode=external -mod=readonly -modcacherw"

    # Build TUI client
    go build -o mkanban ./cmd/mkanban

    # Build daemon
    go build -o mkanbad ./cmd/mkanbad

    # Generate shell completions
    ./mkanban completion bash > mkanban.bash
    ./mkanban completion zsh > mkanban.zsh
    ./mkanban completion fish > mkanban.fish
}

check() {
    cd "${srcdir}/${pkgname}"
    go test -v ./...
}

package() {
    cd "${srcdir}/${pkgname}"

    # Install binaries
    install -Dm755 mkanban "${pkgdir}/usr/bin/mkanban"
    install -Dm755 mkanbad "${pkgdir}/usr/bin/mkanbad"

    # Install systemd service files
    install -Dm644 systemd/mkanbad.service "${pkgdir}/usr/lib/systemd/user/mkanbad.service"
    install -Dm644 systemd/mkanbad@.service "${pkgdir}/usr/lib/systemd/system/mkanbad@.service"

    # Install shell completions
    install -Dm644 mkanban.bash "${pkgdir}/usr/share/bash-completion/completions/mkanban"
    install -Dm644 mkanban.zsh "${pkgdir}/usr/share/zsh/site-functions/_mkanban"
    install -Dm644 mkanban.fish "${pkgdir}/usr/share/fish/vendor_completions.d/mkanban.fish"

    # Install documentation
    install -Dm644 README.md "${pkgdir}/usr/share/doc/${pkgname}/README.md"

    # Install license if available
    if [ -f LICENSE ]; then
        install -Dm644 LICENSE "${pkgdir}/usr/share/licenses/${pkgname}/LICENSE"
    fi
}
