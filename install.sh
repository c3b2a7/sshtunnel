#!/bin/sh

set -u

latest_url=https://api.github.com/repos/c3b2a7/sshtunnel/releases/latest
download_url=https://github.com/c3b2a7/sshtunnel/releases/download
out_prefix=./bin
ask=y
quiet=n
use_go=n
version=""

# Determine default arch/os
os="$(uname -s | tr '[:upper:]' '[:lower:]')"
arch=$(uname -m)

usage() {
  echo "install.sh - A script to help fetch the sshtunnel CLI"
  echo ""
  echo "Flags"
  echo "-----"
  echo "  -h: print this usage message"
  echo "  -q: quiet mode, silence updates to stdout"
  echo "  -a: set the target machine architecture (amd64, arm64, 386)"
  echo "  -s: set the target operating system (linux, macos, freebsd, windows)"
  echo "  -o: installation prefix (default: ./bin)"
  echo "  -v: install the specified version (default: latest)"
  echo "  -y: accept defaults, don't ask before executing commands"
  echo "  -g: build from source using go"
}

latest_tag() {
  release_url=$1

  curl -sfL "$release_url" | grep tag_name | awk '{ print $2 }' | sed 's/[",]//g'
}

install_asset() {
  asset_filepath=$1
  destination=$2
  archive_binary=$3
  destination_binary=$4

  print "Extracting release to $destination/$destination_binary"
  archive_dir=$(dirname "$asset_filepath")
  (cd "$archive_dir" && unpack "$asset_filepath") || return 1
  install "$archive_dir/$archive_binary" "$destination/$destination_binary" || return 1
}

unpack() {
  archive=$1
  case "$archive" in
    *.tar.gz | *.tgz) tar -xzf "$archive" ;;
    *.zip) unzip -q "$archive" ;;
    *) return 1 ;;
  esac
}

download_release() {
  releases_url=$1
  version=$2
  archive=$3
  output_filepath=$4

  curl -sfL "$releases_url/$version/$archive" -o "$output_filepath"
}

install_release() {
  releases_url=$1
  version=$2
  archive=$3
  archive_binary=$4
  destination=$5
  destination_binary=$6

  tmp_dir=$(mktemp -d) || err "Unable to create temporary directory"
  asset_filepath="$tmp_dir/$archive"
  download_release "$releases_url" "$version" "$archive" "$asset_filepath" || {
    rm -rf "$tmp_dir"
    err "Unable to download $archive"
  }
  install_asset "$asset_filepath" "$destination" "$archive_binary" "$destination_binary" || {
    rm -rf "$tmp_dir"
    err "Unable to install $archive"
  }
  rm -rf "$tmp_dir"
  chmod +x "$destination/$destination_binary"
}

go_install() {
  source_dir=$1
  destination=$2
  executable=$3

  print "Copying executable to $destination/$executable"
  cp "$source_dir/$executable" "$destination/$executable"
}

print() {
  if [ "$quiet" = "n" ]; then
    echo "$@"
  fi
}

err() {
  echo "$@" >&2
  exit 1
}

while getopts "hyqa:s:o:v:g" arg "$@"; do
  case "$arg" in
    h)
      usage
      exit 0
      ;;
    y)
      ask=n
      ;;
    q)
      quiet=y
      ;;
    a)
      arch=$OPTARG
      ;;
    s)
      os=$OPTARG
      ;;
    o)
      out_prefix=$OPTARG
      ;;
    v)
      version=$OPTARG
      ;;
    g)
      use_go=y
      ;;
    *) ;;
  esac
done

# Fix arch names
case "$arch" in
  x86_64) arch=amd64 ;;
  aarch64) arch=arm64 ;;
  i386 | i686) arch=386 ;;
  *) ;;
esac

# Fix os names
case "$os" in
  macos) os=darwin ;;
  mingw* | msys* | cygwin* | win*) os=windows ;;
  *) ;;
esac

binary=sshtunnel
archive_member=sshtunnel-$os-$arch
archive_name=sshtunnel-$os-$arch.tar.gz
case "$os-$arch" in
  darwin-amd64 | darwin-arm64) ;;
  linux-amd64 | linux-arm64) ;;
  freebsd-amd64 | freebsd-arm64) ;;
  windows-amd64)
    binary=sshtunnel.exe
    archive_member=sshtunnel-win64.exe
    archive_name=sshtunnel-win64.zip
    ;;
  windows-386)
    binary=sshtunnel.exe
    archive_member=sshtunnel-win32.exe
    archive_name=sshtunnel-win32.zip
    ;;
  *)
    print "No prebuilt executables are available for $os-$arch"
    print "Attempting to build from source"
    use_go=y
    ;;
esac

# Get latest version if none was specified
if test -z "$version"; then
  version="$(latest_tag "$latest_url")"
fi

if [ "$ask" = "y" ] && [ ! -t 0 ]; then
  if [ ! -t 1 ]; then
    err "Unable to run interactively. Run with -y to accept defaults, -h for additional options"
  fi
fi

if [ "$use_go" = "y" ]; then
  go_bin=$(go env GOBIN)
  if [ "$go_bin" = "" ]; then
    go_path=$(go env GOPATH)
    if [ "$go_path" != "" ]; then
      go_bin="$go_path/bin"
    fi
  fi
  if [ "$go_bin" = "" ]; then
    go_bin="$HOME/go/bin"
  fi
fi

if [ "$ask" = "y" ]; then
  echo "Confirm installation:"
  echo "  Version: $version"
  echo "  OS: $os"
  echo "  Arch: $arch"
  echo "  Destination: $out_prefix/$binary"
  echo "  Build from source: $use_go"
  echo "Proceed? [y/N]:"
  read -r reply </dev/tty
else
  reply=y
fi
if [ "$reply" = "y" ] || [ "$reply" = "Y" ] || [ "$reply" = "yes" ]; then
  if [ ! -d "$out_prefix" ]; then
    print "Creating directory $out_prefix"
    mkdir -p "$out_prefix"
  fi
  if [ "$use_go" = "y" ]; then
    print "Installing using go install"
    GOOS=$os GOARCH=$arch go install "github.com/c3b2a7/sshtunnel@$version" || err "Unable to install from source, make sure go is installed"
    go_install "$go_bin" "$out_prefix" "$binary"
    exit 0
  else
    install_release "$download_url" "$version" "$archive_name" "$archive_member" "$out_prefix" "$binary"
  fi
  print "sshtunnel executable installed to $out_prefix/$binary"
else
  err "Exiting"
fi
