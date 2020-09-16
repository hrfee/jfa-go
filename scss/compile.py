#!/usr/bin/env python3
import sass
import subprocess
import shutil
import os
import argparse
from pathlib import Path

parser = argparse.ArgumentParser()

parser.add_argument(
    "-y", "--yes", help="use assumed node bin directory.", action="store_true"
)


def runcmd(cmd):
    if os.name == "nt":
        return subprocess.check_output(cmd, shell=True)
    proc = subprocess.Popen(cmd.split(), stdout=subprocess.PIPE)
    return proc.communicate()


local_path = Path(__file__).resolve().parent
out = runcmd("npm bin")

try:
    node_bin = Path(out[0].decode("utf-8").rstrip())
except:
    node_bin = Path(out.decode("utf-8").rstrip())

args = parser.parse_args()

if not args.yes:
    print(f'assuming npm bin directory "{node_bin}". Is this correct?')
    if input("[yY/nN]: ").lower() == "n":
        node_bin = local_path.parent / "node_modules" / ".bin"
        print(f'this? "{node_bin}"')
        if input("[yY/nN]: ").lower() == "n":
            node_bin = input("input bin directory: ")

for bsv in [d for d in local_path.iterdir() if "bs" in d.name]:
    scss = [(bsv / f"{bsv.name}-jf.scss"), (bsv / f"{bsv.name}.scss")]
    css = [(bsv / f"{bsv.name}-jf.css"), (bsv / f"{bsv.name}.css")]
    min_css = [(bsv.parents[1] / "data" / "static" / f"{bsv.name}-jf.css"), (bsv.parents[1] / "data" / "static" / f"{bsv.name}.css")]
    for i in range(2):
        with open(css[i], "w") as f:
            f.write(
                sass.compile(
                    filename=str(scss[i].resolve()), output_style="expanded", precision=6
                )
            )
        if css[i].exists():
            print(f"{scss[i].name}: Compiled.")
            # postcss only excepts forwards slashes? weird.
            cssPath = str(css[i].resolve())
            if os.name == "nt":
                cssPath = cssPath.replace("\\", "/")
            runcmd(
                f'{str((node_bin / "postcss").resolve())} {cssPath} --replace --use autoprefixer'
            )
            print(f"{scss[i].name}: Prefixed.")
            runcmd(
                f'{str((node_bin / "cleancss").resolve())} --level 1 --format breakWith=lf --output {str(min_css[i].resolve())} {str(css[i].resolve())}'
            )
            if min_css[i].exists():
                print(f"{scss[i].name}: Minified and copied to {str(min_css[i].resolve())}.")

# for v in [("bootstrap", "bs5"), ("bootstrap4", "bs4")]:
#     new_path = str((local_path.parent / "data" / "static" / (v[1] + ".css")).resolve())
#     shutil.copy(
#         str(
#             (
#                 local_path.parent
#                 / "node_modules"
#                 / v[0]
#                 / "dist"
#                 / "css"
#                 / "bootstrap.min.css"
#             ).resolve()
#         ),
#         new_path,
#     )
#     print(f"Copied {v[1]} to {new_path}")
