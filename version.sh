#!/bin/bash

sed "s/{.VERSION}/$(cat version)/g" common/version.go.template > common/version.go
sed "s/{.VERSION}/$(cat version)/g" docs_src/config.toml.template > docs_src/config.toml