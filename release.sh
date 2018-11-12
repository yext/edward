#!/bin/bash

if [ -z "$(git status --porcelain)" ]; then 
    # Working directory clean
    echo "No changes to commit"
else 
    # Uncommitted changes
    git add .
    git commit -m "Automated commit for release"
fi
git push
git checkout master
git pull
git merge develop
git tag v$(cat VERSION)
git push origin
git push origin v$(cat VERSION)