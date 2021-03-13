# Since go doesn't order its json, this script adds ordered lists
# of section/setting names for the settings tab to use.
import json, argparse

parser = argparse.ArgumentParser()
parser.add_argument("-i", "--input", help="input config base from jf-accounts")
parser.add_argument("-o", "--output", help="output config base for jfa-go")

args = parser.parse_args()

with open(args.input, 'r') as f:
    config = json.load(f)

newconfig = {"sections": {}, "order": []}

for sect in config["sections"]:
    newconfig["order"].append(sect)
    newconfig["sections"][sect] = {}
    newconfig["sections"][sect]["order"] = []
    newconfig["sections"][sect]["meta"] = config["sections"][sect]["meta"]
    newconfig["sections"][sect]["settings"] = {}
    for setting in config["sections"][sect]["settings"]:
        newconfig["sections"][sect]["order"].append(setting)
        newconfig["sections"][sect]["settings"][setting] = config["sections"][sect]["settings"][setting]

with open(args.output, 'w') as f:
    f.write(json.dumps(newconfig, indent=4))


