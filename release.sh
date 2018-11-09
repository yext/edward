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
echo git checkout master
echo git pull
echo git merge develop
echo git tag v$(cat VERSION)
echo git push origin v$(cat VERSION)