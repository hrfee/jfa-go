###  fixconfig

Python's `json` library retains the order of data in a JSON file, which meant settings sent to the web page would be in the right order. Go's `encoding/json` and maps do not retain order, so `enumerate/enumerate_config.py` opens the json file, and for each section, adds an "order" array which tells the web page in which order to display settings. 
Specify the input and output files with `-i` and `-o` respectively.
