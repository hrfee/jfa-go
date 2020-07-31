## fixconfig

Python's `json` library retains the order of data in a JSON file, which meant settings sent to the web page would be in the right order. Go's `encoding/json` and maps do not retain order, so this script opens the json file, and for each section, adds an "order" list which tells the web page in which order to display settings. 

Place the config base at `./config-base.json`, run `python fixconfig.py`, and the new config base will be stored at `./ordered-config-base.json`.
