import json, argparse

parser = argparse.ArgumentParser()
parser.add_argument("-i", "--input", help="input config base from jf-accounts")
parser.add_argument("-o", "--output", help="output config base for jfa-go")

args = parser.parse_args()

with open(args.input, 'r') as f:
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

with open(args.output, 'w') as f:
    f.write(json.dumps(newconfig, indent=4))


