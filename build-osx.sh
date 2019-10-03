#!/bin/bash

PLATFORM=$1
if [ "$PLATFORM" == "win" ]; then
	go get github.com/Sirupsen/logrus
	go get github.com/pkg/errors
	go get github.com/urfave/cli
fi



nowDate=$(date +"%Y_%m_%d")
nowTime=$(date +"%H_%M_%S")
commitHash=$(git rev-parse --short HEAD)
versionString="$nowDate.$nowTime.0.$commitHash"
echo $versionString

versionOld=$(grep "const LomoUpdateVersion" main.go)
echo "old verion: $versionOld"
sed -i ".bak" -E "s/[[:digit:]]{4}_[[:digit:]]{2}_[[:digit:]]{2}\.[[:digit:]]{2}_[[:digit:]]{2}_[[:digit:]]{2}\.0\.[a-zA-Z0-9]{7}/$versionString/g" main.go
versionNew=$(grep "const LomoUpdateVersion" main.go)
echo "new verion: $versionNew"

if [ "$PLATFORM" == "win" ]; then
	go build -o lomoupg.exe
	zip -r lomoUpdateWin.zip lomoupg.exe
else
	go build -o lomoupg
	zip -r lomoUpdateOSX.zip lomoupg
	shasum -a256 lomoUpdateOSX.zip
fi


