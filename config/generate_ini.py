# Generates config file
import configparser
import json
import argparse
from pathlib import Path

parser = argparse.ArgumentParser()
parser.add_argument("-i", "--input", help="input config base from jf-accounts")
parser.add_argument("-o", "--output", help="output ini")


def generate_ini(base_file, ini_file, version):
    """
    Generates .ini file from config-base file.
    """
    with open(Path(base_file), "r") as f:
        config_base = json.load(f)

    ini = configparser.RawConfigParser(allow_no_value=True)

    for section in config_base:
        ini.add_section(section)
        for entry in config_base[section]:
            if "description" in config_base[section][entry]:
                ini.set(section, "; " + config_base[section][entry]["description"])
            if entry != "meta":
                value = config_base[section][entry]["value"]
                if isinstance(value, bool):
                    value = str(value).lower()
                else:
                    value = str(value)
                ini.set(section, entry, value)

    ini["jellyfin"]["version"] = version
    ini["jellyfin"]["device_id"] = ini["jellyfin"]["device_id"].replace(
        "{version}", version
    )

    with open(Path(ini_file), "w") as config_file:
        ini.write(config_file)
    return True


args = parser.parse_args()

print(generate_ini(base_file=args.input,
                   ini_file=args.output,
                   version="0.1.0"))
