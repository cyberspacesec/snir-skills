#!/usr/bin/env sh
set -eu

repo="${SNIR_REPO:-cyberspacesec/snir-skills}"
version="${SNIR_VERSION:-latest}"
prefix="${SNIR_PREFIX:-/usr/local/bin}"
binary_name="${SNIR_BINARY:-snir}"

need_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "missing required command: $1" >&2
    exit 1
  fi
}

detect_os() {
  case "$(uname -s)" in
    Linux) echo "Linux" ;;
    Darwin) echo "Darwin" ;;
    FreeBSD) echo "Freebsd" ;;
    OpenBSD) echo "Openbsd" ;;
    NetBSD) echo "Netbsd" ;;
    *) echo "unsupported OS: $(uname -s)" >&2; exit 1 ;;
  esac
}

detect_arch() {
  arch="$(uname -m)"
  case "$arch" in
    x86_64|amd64) echo "x86_64" ;;
    aarch64|arm64) echo "arm64" ;;
    i386|i686) echo "i386" ;;
    armv5*) echo "armv5" ;;
    armv6*) echo "armv6" ;;
    armv7*) echo "armv7" ;;
    mips) echo "mips" ;;
    mipsel) echo "mipsle" ;;
    mips64) echo "mips64" ;;
    mips64el) echo "mips64le" ;;
    ppc64le) echo "ppc64le" ;;
    riscv64) echo "riscv64" ;;
    s390x) echo "s390x" ;;
    *) echo "unsupported architecture: $arch" >&2; exit 1 ;;
  esac
}

latest_version() {
  release_json="$(curl -fsSL "https://api.github.com/repos/${repo}/releases/latest")" || return 1
  printf '%s\n' "$release_json" |
    sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' |
    head -n 1
}

install_binary() {
  src="$1"
  dst="${prefix}/${binary_name}"

  if [ -w "$prefix" ]; then
    install -m 0755 "$src" "$dst"
  elif command -v sudo >/dev/null 2>&1; then
    sudo install -m 0755 "$src" "$dst"
  else
    echo "cannot write to $prefix and sudo is unavailable" >&2
    exit 1
  fi
}

need_cmd curl
need_cmd tar
need_cmd sed
need_cmd install

if [ "$version" = "latest" ]; then
  version="$(latest_version)"
fi

if [ -z "$version" ]; then
  echo "could not determine snir release version" >&2
  exit 1
fi

os="$(detect_os)"
arch="$(detect_arch)"
asset="snir-skills_${os}_${arch}.tar.gz"
url="https://github.com/${repo}/releases/download/${version}/${asset}"
tmpdir="$(mktemp -d)"

cleanup() {
  rm -rf "$tmpdir"
}
trap cleanup EXIT INT TERM

echo "downloading $url"
curl -fL -o "$tmpdir/snir.tar.gz" "$url"
tar xzf "$tmpdir/snir.tar.gz" -C "$tmpdir" snir
install_binary "$tmpdir/snir"

echo "installed ${binary_name} to ${prefix}"
"${prefix}/${binary_name}" version
