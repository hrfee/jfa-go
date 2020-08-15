import json

with open("config-formatted.json", "r") as f:
    config = json.load(f)

indent = 0


def writeln(ln):
    global indent
    if "}" in ln and "{" not in ln:
        indent -= 1
    s.write(("\t" * indent) + ln + "\n")
    if "{" in ln and "}" not in ln:
        indent += 1


with open("configStruct.go", "w") as s:
    writeln("package main")
    writeln("")
    writeln("type Metadata struct{")
    writeln('Name string `json:"name"`')
    writeln('Description string `json:"description"`')
    writeln("}")
    writeln("")
    writeln("type Config struct{")
    if "order" in config:
        writeln('Order []string `json:"order"`')
    for section in [x for x in config.keys() if x != "order"]:
        title = "".join([x.title() for x in section.split("_")])
        writeln(title + " struct{")
        if "order" in config[section]:
            writeln('Order []string `json:"order"`')
        if "meta" in config[section]:
            writeln('Meta Metadata `json:"meta"`')
        for setting in [
            x for x in config[section].keys() if x != "order" and x != "meta"
        ]:
            name = "".join([x.title() for x in setting.split("_")])
            writeln(name + " struct{")
            writeln('Name string `json:"name"`')
            writeln('Required bool `json:"required"`')
            writeln('Restart bool `json:"requires_restart"`')
            writeln('Description string `json:"description"`')
            writeln('Type string `json:"type"`')
            dt = config[section][setting]["type"]
            if dt == "select":
                dt = "string"
                writeln('Options []string `json:"options"`')
            elif dt == "number":
                dt = "int"
            elif dt != "bool":
                dt = "string"
            writeln(f'Value {dt} `json:"value" cfg:"{setting}"`')
            writeln("} " + f'`json:"{setting}" cfg:"{setting}"`')
        writeln("} " + f'`json:"{section}"`')
    writeln("}")
