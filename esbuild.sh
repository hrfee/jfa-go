#!/usr/bin/bash
# set +e
# npx tsc -p ts/
# set -e
npx esbuild ts/* --outdir=data/static --minify
