#!/usr/bin/env python3
import sass
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

for bsv in [d for d in local_path.iterdir() if "bs" in d.name]:
    scss = [(bsv / f"{bsv.name}-jf.scss"), (bsv / f"{bsv.name}.scss")]
    css = [(bsv / f"{bsv.name}-jf.css"), (bsv / f"{bsv.name}.css")]
    min_css = [
        (bsv.parents[1] / "data" / "static" / f"{bsv.name}-jf.css"),
        (bsv.parents[1] / "data" / "static" / f"{bsv.name}.css"),
    ]
    for i in range(2):
        with open(css[i], "w") as f:
            f.write(
                sass.compile(
                    filename=str(scss[i].resolve()),
                    output_style="expanded",
                    precision=6,
                    omit_source_map_url=True,
                )
            )
        if css[i].exists():
            print(f"{scss[i].name}: Compiled.")
            # postcss only excepts forwards slashes? weird.
            cssPath = str(css[i].resolve())
            if os.name == "nt":
                cssPath = cssPath.replace("\\", "/")
            runcmd(f"npx postcss {cssPath} --replace --use autoprefixer")
            print(f"{scss[i].name}: Prefixed.")
            runcmd(
                f"npx cleancss --level 1 --format breakWith=lf --output {str(min_css[i].resolve())} {str(css[i].resolve())}"
            )
            if min_css[i].exists():
                print(
                    f"{scss[i].name}: Minified and copied to {str(min_css[i].resolve())}."
                )
