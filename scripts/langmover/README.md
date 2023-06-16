# *langmover*

* Makes moving strings between language files a little easier.

# Usage

You'll need to create a template file. See example `template.json`:
```json
{
    "meta": {
        "explanation": "values here can either be folder, folder:section, or folder:section:subkey, and then either nothing, or /keyname. It all depends on whether the sections and keys match up, or if you want to pull a plural/singular only or not."
    },
    "strings": {
        "inviteInfiniteUsesWarning": "admin", // Resolves to admin/strings/inviteInfiniteUsesWarning
        "emailAddress": "form:strings/emailAddress", // Resolves to form/strings/emailAddress
        "modifySettingsFor": "admin:quantityStrings:plural/", // Resolves to admin/quantityStrings/modifySettingsFor/plural
        "deleteNUsers": "admin:quantityStrings:singular/deleteNUsers" // Resolves to admin/quantityStrings/deleteNUsers/singular
    },
    "quantityStrings": {
        "reEnableUsers": "admin" // Resolves to admin/quantityStrings/reEnableUsers
    }

}
```


Args:
* `--source`: Source `lang/` directory. **Always run on a copy, to avoid data loss**
* `--template`: Template JSON file.
* `--output`: Output directory. Will be filled with lang files (e.g. "en-us.json", "fa-ir.json", ...).
* `--extract`: Passing will remove the templated strings from their source file. **Modifies the source directory**.


