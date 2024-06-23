#!/bin/sh

set -u

latest_url=https://api.github.com/repos/c3b2a7/sshtunnel/releases/latest
download_url=https://github.com/c3b2a7/sshtunnel/releases/download
out_prefix=/usr/local/bin
ask=y
quiet=n
use_go=n
version=""

# Determine default arch/os
os=$(uname -s | awk '{print tolower($0)}')
arch=$(uname -m)

usage() {
  echo "get-sshtunnel.sh - A script to help fetch the sshtunnel CLI"
  echo ""
  echo "Flags"
  echo "-----"
  echo "  -h: print this usage message"
  echo "  -q: quiet mode, silence updates to stdout"
  echo "  -a: set the target machine architecture (amd64, arm64)"
  echo "  -s: set the target operating system (linux, macos)"
  echo "  -o: installation prefix (default: /usr/local/bin)"
  echo "  -y: accept defaults, don't ask before executing commands"
  echo "  -g: build from source using go"
}

latest_tag() {
  curl -s $latest_url | grep tag_name | awk '{ print $2 }' | sed 's/[",]//g'
}

untar() {
  print "Extracting release to $out_prefix/sshtunnel"
  $_sudo sh -c "tar -xzO sshtunnel-$os-$arch > \"$out_prefix/sshtunnel\""
  $_sudo chmod +x "$out_prefix/sshtunnel"
}

go_install_exe() {
  print "Copying executable to $out_prefix/sshtunnel"
  $_sudo cp "$go_bin"/sshtunnel "$out_prefix"/sshtunnel
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
        *)
            ;;
        esac
done


_user=$(whoami)
if [ "$_user" = "root" ]; then
  _sudo=""
else
  _sudo=$(which sudo)
  case $out_prefix in
  $HOME/*)
    _sudo=""
    ;;
  *)
    if [ "$_sudo" = "" ]; then
      echo "No sudo installation found, but needed to install into $out_prefix" && exit 1
    fi
    ;;
  esac
fi

# Fix arch names
case "$arch" in
x86_64)
  arch=amd64 ;;
aarch64)
  arch=arm64 ;;
*)
  ;;
esac

# Fix os names
case "$os" in
macos)
  os=darwin;;
*)
  ;;
esac

case "$os-$arch" in
  darwin-amd64|darwin-arm64) ;;
  linux-amd64|linux-arm64) ;;
  *)
    print "No prebuilt executables are available for $os-$arch"
    print "Attempting to build from source"
    use_go=y
esac

# Get latest version if none was specified
if test -z "$version"
then
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
    go_bin=$(go env GOPATH)
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
  if [ "$_sudo" != "" ]; then
    echo "  Sudo: $_sudo"
  fi
  echo "  Destination: $out_prefix/sshtunnel"
  echo "  Build from source: $use_go"
  echo "Proceed? [y/N]:"
  read -r reply < /dev/tty
else
  reply=y
fi
if [ "$reply" = "y" ] || [ "$reply" = "Y" ] || [ "$reply" = "yes" ]; then
  if [ ! -d "$out_prefix" ]; then
    print "Creating directory $out_prefix"
    $_sudo mkdir -p "$out_prefix"
  fi
  if [ "$use_go" = "y" ]; then
    print "Installing using go install"
    GOOS=$os GOARCH=$arch go install "github.com/c3b2a7/sshtunnel@$version" || err "Unable to install from source, make sure go is installed"
    go_install_exe
    exit 0
  else
    curl -L -s "$download_url/$version/sshtunnel-$os-$arch.tar.gz" | untar
  fi
  print "sshtunnel executable installed to $out_prefix/sshtunnel"
else
  err "Exiting"
fi
