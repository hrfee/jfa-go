# Generates config file
import configparser
import json
import argparse
from pathlib import Path

def fix_description(desc):
    return "; " + desc.replace("\n", "\n; ")

def generate_ini(base_file, ini_file):
    """
    Generates .ini file from config-base file.
    """
    with open(Path(base_file), "r") as f:
        config_base = json.load(f)

    ini = configparser.RawConfigParser(allow_no_value=True)

    for section in config_base["sections"]:
        ini.add_section(section)
        if "meta" in config_base["sections"][section]:
            ini.set(section, fix_description(config_base["sections"][section]["meta"]["description"]))
        for entry in config_base["sections"][section]["settings"]:
            if config_base["sections"][section]["settings"][entry]["type"] == "note":
                continue
            if "description" in config_base["sections"][section]["settings"][entry]:
                ini.set(section, fix_description(config_base["sections"][section]["settings"][entry]["description"]))
            value = config_base["sections"][section]["settings"][entry]["value"]
            if isinstance(value, bool):
                value = str(value).lower()
            else:
                value = str(value)
            ini.set(section, entry, value)

    with open(Path(ini_file), "w") as config_file:
        ini.write(config_file)
    return True


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("-i", "--input", help="input config base from jf-accounts")
    parser.add_argument("-o", "--output", help="output ini")

    args = parser.parse_args()

    print(generate_ini(base_file=args.input, ini_file=args.output))
