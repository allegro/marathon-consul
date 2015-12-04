#!/usr/bin/env bash

export PATH=./bin:$PATH

hash consul 2>/dev/null || {
    echo "Installing consul..."

    os=`uname -s | awk '{print tolower($0)}'`
    arch=`uname -m | awk '{print tolower($0)}'`

    if [ $os = 'linux' ] ; then
        if [ $arch = 'x86_64' ] ; then
            arch="amd64"
        else
            arch="386"
        fi
    elif [ $os = 'darwin' ] ; then
        arch="amd64"
    else
        os="windows"
        arch="386"
    fi
    archive="0.5.2_${os}_${arch}.zip"

    mkdir -p bin
    curl -OLs https://dl.bintray.com/mitchellh/consul/$archive
    unzip -q "$archive" -d bin
    rm "$archive"
    consul --version
}

