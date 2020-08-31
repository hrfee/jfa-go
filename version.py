import subprocess
import sys
import os
try:
    version = sys.argv[1].replace('v', '')
except IndexError:
    version = "git"

commit = subprocess.check_output("git rev-parse --short HEAD".split()).decode("utf-8").rstrip()

file = f'package main; const VERSION = "{version}"; const COMMIT = "{commit}";'

try:
    writeto = sys.argv[2]
except IndexError:
    writeto = "version.go"

with open(writeto, 'w') as f:
    f.write(file)

