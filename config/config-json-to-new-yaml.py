from ruamel.yaml import YAML
import json
from pathlib import Path
import sys
yaml = YAML()

# c = yaml.load(Path(sys.argv[len(sys.argv)-1]))
with open(sys.argv[len(sys.argv)-1], 'r') as f:
    c = json.load(f)

c.pop("order")

c1 = c.copy()
c1["sections"] = []
for section in c["sections"]:
    codeSection = { "section": section }
    s = codeSection | c["sections"][section]
    s.pop("order")
    c1["sections"].append(s)

c2 = c.copy()
c2["sections"] = []

for section in c1["sections"]:
    sArray = []
    for setting in section["settings"]:
        codeSetting = { "setting": setting }
        s = codeSetting | section["settings"][setting]
        sArray.append(s)

    section["settings"] = sArray
    c2["sections"].append(section)


yaml.dump(c2, sys.stdout)
