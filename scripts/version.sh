#!/bin/bash
JFA_GO_VERSION=$(git describe --exact-match HEAD 2> /dev/null || echo 'vgit')
JFA_GO_VERSION="$(echo $JFA_GO_VERSION | sed 's/v//g')" $@
