#!/bin/bash

PUBLISH_DIR=`mktemp -d`
PUBLISH_DIR=$PUBLISH_DIR/doc_test

pushd docs_src > /dev/null
hugo --quiet -d $PUBLISH_DIR
popd > /dev/null

COUNT="$(git diff --no-index docs $PUBLISH_DIR | wc -l)"
if [ $COUNT != "0" ]; then
  git diff --no-index docs $PUBLISH_DIR | cat
  rm -rf $PUBLISH_DIR
  echo "Documentation not up to date (see above diff). Run hugo in docs_src to build."
  exit 2
else
  echo "Documentation is up to date!"
  rm -rf $PUBLISH_DIR
fi
