import json
import sys

sectionSchema = {}
metaSchema = {}
settingSchema = {}
typeValues = {}

# c = yaml.load(Path(sys.argv[len(sys.argv)-1]))
with open(sys.argv[len(sys.argv)-1], 'r') as f:
    c = json.load(f)

for section in c["sections"]:
    for key in c["sections"][section]:
        sectionSchema[key] = True

    for key in c["sections"][section]["meta"]:
        metaSchema[key] = c["sections"][section]["meta"][key]

    for setting in c["sections"][section]["settings"]:
        for field in c["sections"][section]["settings"][setting]:
            settingSchema[field] = c["sections"][section]["settings"][setting][field]
        typeValues[c["sections"][section]["settings"][setting]["type"]] = True

print("Section Content:")
for v in sectionSchema:
    print(v)
print("---")
print("Meta Schema")
for v in metaSchema:
    print(v, "=", type(metaSchema[v]))
print("---")
print("Setting Schema")
for v in settingSchema:
    print(v, "=", type(settingSchema[v]))
print("---")
print("Possible Types")
for v in typeValues:
    print(v)

