import subprocess
import shutil
import os
from pathlib import Path

def runcmd(cmd):
    if os.name == "nt":
        return subprocess.check_output(cmd, shell=True)
    proc = subprocess.Popen(cmd.split(), stdout=subprocess.PIPE)
    return proc.communicate()

local_path = Path(__file__).resolve().parent
out = runcmd("npm bin")

try:
    node_bin = Path(out[0].decode('utf-8').rstrip())
except:
    node_bin = Path(out.decode('utf-8').rstrip())

print(f"assuming npm bin directory \"{node_bin}\". Is this correct?")
if input("[yY/nN]: ").lower() == "n":
    node_bin = local_path.parent / 'node_modules' / '.bin'
    print(f"this? \"{node_bin}\"")
    if input("[yY/nN]: ").lower() == "n":
        node_bin = input("input bin directory: ")

for mjml in [f for f in local_path.iterdir() if f.is_file() and 'mjml' in f.suffix]:
    print(f'Compiling {mjml.name}')
    fname = mjml.with_suffix(".html")
    runcmd(f'{str(node_bin / "mjml")} {str(mjml)} -o {str(fname)}')
    if fname.is_file():
        print('Done.')

html = [f for f in local_path.iterdir() if f.is_file() and 'html' in f.suffix]

output = local_path.parent / 'data'

for f in html:
    shutil.copy(str(f),
                str(output / f.name))
    print(f'Copied {f.name} to {str(output / f.name)}')
    txtfile = f.with_suffix('.txt')
    if txtfile.is_file():
        shutil.copy(str(txtfile),
                    str(output / txtfile.name))
        print(f'Copied {txtfile.name} to {str(output / txtfile.name)}')
    else:
        print(f'Warning: {txtfile.name} does not exist. Text versions of emails should be supplied.')

