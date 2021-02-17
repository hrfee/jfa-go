#!/usr/bin/python
import shutil
import sys
from pathlib import Path

external = ["false", "external", "no", "n"]

with open("embed.go", "w") as f:
    choice = ""
    try:
        choice = sys.argv[1]
    except IndexError:
        pass
    folder = Path("embed")
    if choice in external:
        embed = False
        shutil.copy(folder / "external.go", "embed.go")
        print("Embedding disabled. \"data\" must be placed alongside the executable.")
    else:
        shutil.copy(folder / "internal.go", "embed.go")
        print("Embedding enabled.")

