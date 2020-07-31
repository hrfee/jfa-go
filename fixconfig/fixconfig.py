import json

with open('config-base.json', 'r') as f:
    config = json.load(f)

newconfig = {"order": []}

for sect in config:
    newconfig["order"].append(sect)
    newconfig[sect] = {}
    newconfig[sect]["order"] = []
    newconfig[sect]["meta"] = config[sect]["meta"]
    for setting in config[sect]:
        if setting != "meta":
            newconfig[sect]["order"].append(setting)
            newconfig[sect][setting] = config[sect][setting]

with open('ordered-config-base.json', 'w') as f:
    f.write(json.dumps(newconfig, indent=4))


