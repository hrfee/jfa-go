# Quick script to scrape custom content names, variables and conditionals. The decision to generate vars and conds dynamically from the included plaintext emails, then bodge extra variables on top was stupid, and is only done -once- before getting stored in the DB indefinitely, meaning new variables can't easily be added. Output of this will be coalesced into a predefined list included with the software.
import requests, json

content = requests.get("http://localhost:8056/config/emails?lang=en-gb&filter=user").json()

out = {}

for key in content:
    resp = requests.get("http://localhost:8056/config/emails/"+key)
    out[key] = resp.json()
    del out[key]["html"]
    del out[key]["plaintext"]
    del out[key]["content"]

print(json.dumps(out, indent=4))

