#!/usr/bin/env python3

import subprocess
import os
from pathlib import Path


def runcmd(cmd):
    if os.name == "nt":
        return subprocess.check_output(cmd, shell=True)
    proc = subprocess.Popen(cmd.split(), stdout=subprocess.PIPE)
    return proc.communicate()


print('Installing npm packages')

root_path = Path(__file__).parents[1]
if os.name == 'nt':
    root_path /= 'node_modules'

runcmd('npm install')

if (root_path / 'node_modules' / 'cleancss').exists() or (root_path / 'cleancss').exists():
    print(f'Installed successfully in {str((root_path / "node_modules").resolve())}.')


