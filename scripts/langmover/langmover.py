import json, argparse, os
from pathlib import Path

ROOT = "en-us.json"

# Tree structure: <lang-code.json>/<folder>/<json content>
def generateTree(src: Path):
    tree = {}
    langs = {}
    directories = []

    def readLangFile(path: Path):
        with open(path, 'r') as f:
            content = json.load(f)

        return content


    for directory in os.scandir(src):
        if not directory.is_dir(): continue
        directories.append(directory.name)

        # tree[directory.name] = {}

        for lang in os.scandir(directory.path):
            if not lang.is_file(): continue
            if not ".json" in lang.name: continue
            if lang.name not in langs:
                langs[lang.name] = True
            if lang.name.lower() not in tree:
                tree[lang.name.lower()] = {}

    for lang in langs:
        for directory in directories:
            filepath = Path(src) / Path(directory) / Path(lang)
            if not filepath.exists(): continue
            tree[lang.lower()][directory] = readLangFile(filepath)

    return tree

def parseKey(langTree, currentSection: str, fieldName: str, key: str, extract=False):
    temp = key.split("/")
    loc = temp[0]
    k = ""
    if len(temp) > 1:
        k = temp[1]

    sections = loc.split(":")

    # folder, folder:section or folder:section:subkey
    folder = sections[0]
    section = currentSection
    subkey = None
    if len(sections) > 1:
        section = sections[1]
    if len(sections) > 2:
        subkey = sections[2]


    if k == '':
        k = fieldName

    value = ""
    if folder in langTree and section in langTree[folder] and k in langTree[folder][section]:
        value = langTree[folder][section][k]
        if extract:
            s = langTree[folder][section]
            del s[k]
            langTree[folder][section] = s

    if subkey is not None and subkey in value:
        value = value[subkey]

    return (langTree, folder, value)


def generate(templ: Path, source: Path, output: Path, extract: str, tree):
    with open(templ, "r") as f:
        template = json.load(f)

    if not output.exists():
        output.mkdir()

    for lang in tree:
        out = {}
        for section in template:
            if section == "meta":
                # grab a meta section from the first file we find
                for file in tree[lang]:
                    out["meta"] = tree[lang][file]["meta"]
                    break

                continue

            folder = ""
            out[section] = {}
            for key in template[section]:
                (tree[lang], folder, val) = parseKey(tree[lang], section, key, template[section][key], extract)
                if val != "":
                    out[section][key] = val
            
            # if extract and val != "":
            #     with open(source / folder / lang, "w") as f:
            #         json.dump(modifiedTree[folder], f, indent=4, ensure_ascii=False)

        with open(output / Path(lang), "w") as f:
            json.dump(out, f, indent=4, ensure_ascii=False)

    if extract and extract != "":
        ex = Path(extract)
        if not ex.exists():
            ex.mkdir()

        for lang in tree:
            for folder in tree[lang]:
                if not (ex / folder).exists():
                    (ex / folder).mkdir()
                with open(ex / folder / lang, "w") as f:
                    json.dump(tree[lang][folder], f, indent=4, ensure_ascii=False)



parser = argparse.ArgumentParser()

parser.add_argument("--source", help="source \"lang/\" folder.")
parser.add_argument("--template", help="template file. see template.json for an example of how it works.")
parser.add_argument("--output", help="output directory for new files.")
parser.add_argument("--extract", help="put copies of original files with strings removed in this directory")

args = parser.parse_args()

source = Path(args.source)

tree = generateTree(source)

generate(Path(args.template), source, Path(args.output), args.extract, tree)

# print(json.dumps(tree, sort_keys=True, indent=4))
