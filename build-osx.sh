#!/bin/bash

nowDate=$(date +"%Y_%m_%d")
nowTime=$(date +"%H_%M_%S")
commitHash=$(git rev-parse --short HEAD)
versionString="$nowDate.$nowTime.0.$commitHash"
echo $versionString

versionOld=$(grep "const LomoUpdateVersion" main.go)
echo "old verion: $versionOld"
sed -i.bak -E "s/[[:digit:]]{4}_[[:digit:]]{2}_[[:digit:]]{2}\.[[:digit:]]{2}_[[:digit:]]{2}_[[:digit:]]{2}\.0\.[a-zA-Z0-9]{7}/$versionString/g" main.go
versionNew=$(grep "const LomoUpdateVersion" main.go)
echo "new verion: $versionNew"

CGO_CFLAGS=-mmacosx-version-min=10.10 CGO_LDFLAGS=-mmacosx-version-min=10.10 go build -o lomoupg

zip -r lomoUpdateOSX.zip lomoupg
shasum -a256 lomoUpdateOSX.zip
