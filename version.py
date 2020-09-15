import subprocess
import sys
import os
try:
    version = sys.argv[1].replace('v', '')
except IndexError:
    version = "git"

if version == "auto":
    try:
        version = subprocess.check_output("git describe --exact-match HEAD".split()).decode("utf-8").rstrip().replace('v', '')
    except subprocess.CalledProcessError as e:
        if e.returncode == 128:
            version = "git"

commit = subprocess.check_output("git rev-parse --short HEAD".split()).decode("utf-8").rstrip()

file = f'package main; const VERSION = "{version}"; const COMMIT = "{commit}";'

try:
    writeto = sys.argv[2]
except IndexError:
    writeto = "version.go"

with open(writeto, 'w') as f:
    f.write(file)

