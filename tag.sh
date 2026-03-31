#!/bin/bash
set -e

Release=$(git rev-list v4.0.0.. --count)
git tag -a "v4.0.${Release}" -m "${Release}"
git remote set-url origin 'git@github.com:anxiwuyanzu/openscraper-framework.git'

git push origin --tag
