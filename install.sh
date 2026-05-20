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
  echo "  -y: accept defaults, don't ask before executing commands"
  echo "  -g: build from source using go"
}

latest_tag() {
  curl -s $latest_url | grep tag_name | awk '{ print $2 }' | sed 's/[",]//g'
}

untar() {
  print "Extracting release to $out_prefix/$binary"
  sh -c "tar -xzO \"$binary\" > \"$out_prefix/$binary\""
  chmod +x "$out_prefix/$binary"
}

unzip_release() {
  print "Extracting release to $out_prefix/$binary"
  tmp_file=$(mktemp "${TMPDIR:-/tmp}/sshtunnel.XXXXXX") || err "Unable to create temporary file"
  curl -L -s "$download_url/$version/$archive_name" -o "$tmp_file" || {
    rm -f "$tmp_file"
    err "Unable to download $archive_name"
  }
  sh -c "unzip -p \"$tmp_file\" \"$binary\" > \"$out_prefix/$binary\"" || {
    rm -f "$tmp_file"
    err "Unable to extract $archive_name"
  }
  rm -f "$tmp_file"
  chmod +x "$out_prefix/$binary"
}

go_install_exe() {
  print "Copying executable to $out_prefix/$binary"
  cp "$go_bin"/"$binary" "$out_prefix"/"$binary"
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
if [ "$os" = "windows" ]; then
  binary=sshtunnel.exe
fi
archive_name=sshtunnel-$os-$arch.tar.gz

case "$os-$arch" in
  darwin-amd64 | darwin-arm64) ;;
  linux-amd64 | linux-arm64) ;;
  freebsd-amd64 | freebsd-arm64) ;;
  windows-amd64 | windows-386) ;;
  *)
    print "No prebuilt executables are available for $os-$arch"
    print "Attempting to build from source"
    use_go=y
    ;;
esac

# Get latest version if none was specified
if test -z "$version"; then
  version="$(latest_tag)"
fi

if [ "$ask" = "y" ] && [ ! -t 0 ]; then
  if [ ! -t 1 ]; then
    err "Unable to run interactively. Run with -y to accept defaults, -h for additional options"
  fi
fi

if [ "$use_go" = "y" ]; then
  version="latest"
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
    go_install_exe
    exit 0
  else
    if [ "$os" = "windows" ]; then
      unzip_release
    else
      curl -L -s "$download_url/$version/$archive_name" | untar
    fi
  fi
  print "sshtunnel executable installed to $out_prefix/$binary"
else
  err "Exiting"
fi
