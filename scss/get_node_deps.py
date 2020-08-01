#!/usr/bin/env python3

import subprocess
from pathlib import Path

def runcmd(cmd):
    proc = subprocess.Popen(cmd.split(), stdout=subprocess.PIPE)
    return proc.communicate()

print('Installing npm packages')

root_path = Path(__file__).parents[1]
runcmd(f'npm install --prefix {root_path}')

if (root_path / 'node_modules' / 'cleancss').exists():
    print(f'Installed successfully in {str((root_path / "node_modules").resolve())}.')

