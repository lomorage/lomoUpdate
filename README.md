# lomoUpdate

Used for lomoUpdate release, including Windows and OSX version

# how to build on windows
google choco and install choco
run PS as admin: in PS shell: 

> choco install zip

then run on git-bash shell:

> ./build.sh win


# how to release

First thing need to install hub from https://hub.github.com/

## MacOS
> ./release.sh 

## Windows
> ./release.sh win