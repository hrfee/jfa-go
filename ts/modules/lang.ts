interface Meta {
    name: string;
}

interface quantityString {
    singular: string;
    plural: string;
}

export interface LangFile {
    meta: Meta;
    strings: { [key: string]: string };
    notifications: { [key: string]: string };
    quantityStrings: { [key: string]: quantityString };
}

export class lang implements Lang {
    private _lang: LangFile;
    constructor(lang: LangFile) {
        this._lang = lang;
    }

    get = (sect: string, key: string): string => {
        if (sect == "quantityStrings" || sect == "meta") { return ""; }
        return this._lang[sect][key];
    }

    strings = (key: string): string => this.get("strings", key)
    notif = (key: string): string => this.get("notifications", key)

    var = (sect: string, key: string, ...subs: string[]): string => {
        if (sect == "quantityStrings" || sect == "meta") { return ""; }
        let str = this._lang[sect][key];
        for (let sub of subs) {
            str = str.replace("{n}", sub);
        }
        return str;
    }

    quantity = (key: string, number: number): string => {
        if (number == 1) {
            return this._lang.quantityStrings[key].singular.replace("{n}", ""+number)
        }
        return this._lang.quantityStrings[key].plural.replace("{n}", ""+number);
    }
}






