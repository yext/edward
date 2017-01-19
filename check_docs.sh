#!/bin/bash

PUBLISH_DIR=`mktemp -d`

pushd docs_src
hugo -d $PUBLISH_DIR/edward_doc_test
popd

COUNT="$(git diff --no-index docs $PUBLISH_DIR/edward_doc_test | wc -l)"
rm -rf $PUBLISH_DIR/edward_doc_test
if [ $COUNT != "0" ]; then
  echo "Documentation not up to date. Run hugo in docs_src to build."
  exit 2
fi
