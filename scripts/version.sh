#!/bin/bash
VERSION=$(git describe --exact-match HEAD 2> /dev/null || echo 'vgit')
VERSION="$(echo $VERSION | sed 's/v//g')" $@
