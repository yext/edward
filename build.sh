#!/bin/bash

TAG="${TRAVIS_TAG:-0.0.0}"

rm -rf build
rm -rf dist

gox -output "build/{{.Dir}}_${TAG}_{{.OS}}_{{.Arch}}/{{.Dir}}" -osarch="linux/386 linux/amd64" .
gox -output "build/{{.Dir}}_${TAG}_macOS_64bit/{{.Dir}}" -osarch="darwin/amd64" .

mkdir dist
pushd build
for i in *
do
[ -d "$i" ] && zip -r "../dist/$i.zip" "$i"
done
popd
