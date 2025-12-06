declare var window: GlobalWindow;
import dateParser from "any-date-parser";
import { Temporal } from 'temporal-polyfill';

export function toDateString(date: Date): string {
    const locale = window.language || (window as any).navigator.userLanguage || window.navigator.language;
    const t12 = document.getElementById("lang-12h") as HTMLInputElement;
    const t24 = document.getElementById("lang-24h") as HTMLInputElement;
    let args1 = {};
    let args2: Intl.DateTimeFormatOptions = {
        hour: "2-digit",
        minute: "2-digit"
    };
    if (t12 && t24) {
        if (t12.checked) {
            args1["hour12"] = true;
            args2["hour12"] = true;
        } else if (t24.checked) {
            args1["hour12"] = false;
            args2["hour12"] = false;
        }
    }
    return date.toLocaleDateString(locale, args1) + " " + date.toLocaleString(locale, args2);
}

export const parseDateString = (value: string): ParsedDate => {
    let out: ParsedDate = {
        text: value,
        // Used just to tell use what fields the user passed.
        attempt: dateParser.attempt(value),
        // note Date.fromString is also provided by dateParser.
        date: (Date as any).fromString(value) as Date
    };
    if (("invalid" in (out.date as any))) {
        out.invalid = true;
    } else {
        // getTimezoneOffset returns UTC - Timezone, so invert it to get distance from UTC -to- timezone.
        out.attempt.offsetMinutesFromUTC = -1 * out.date.getTimezoneOffset();
    }
    // Month in Date objects is 0-based, so make our parsed date that way too
    if ("month" in out.attempt) out.attempt.month -= 1;
    return out;
}

// DateCountdown sets the given el's textContent to the time till the given date (unixSeconds), updating
// every minute. It returns the timeout, so it can be later removed with clearTimeout if desired.
export function DateCountdown(el: HTMLElement, unixSeconds: number): ReturnType<typeof setTimeout> {
    let then = Temporal.Instant.fromEpochMilliseconds(unixSeconds * 1000);
    const toString = (): string => {
        let out = "";
        let now = Temporal.Now.instant();
        let nowPlain = Temporal.Now.plainDateTimeISO();
        let diff = now.until(then).round({
            largestUnit: "years",
            smallestUnit: "minutes",
            relativeTo: nowPlain
        });
        // FIXME: I'd really like this to be localized, but don't know of any nice solutions.
        const fields = [diff.years, diff.months, diff.days, diff.hours, diff.minutes];
        const abbrevs = ["y", "mo", "d", "h", "m"];
        for (let i = 0; i < fields.length; i++) {
            if (fields[i]) {
                out += ""+fields[i] + abbrevs[i] + " ";
            }
        }
        return out.slice(0, -1);
    };
    const update = () => {
        el.textContent = toString();
    };
    update();
    return setTimeout(update, 60000);
}

export const _get = (url: string, data: Object, onreadystatechange: (req: XMLHttpRequest) => void, noConnectionError: boolean = false): void => {
    let req = new XMLHttpRequest();
    if (window.pages) { url = window.pages.Base + url; }
    req.open("GET", url, true);
    req.responseType = 'json';
    req.setRequestHeader("Authorization", "Bearer " + window.token);
    req.setRequestHeader('Content-Type', 'application/json; charset=UTF-8');
    req.onreadystatechange = () => {
        if (req.status == 0) {
            if (!noConnectionError) window.notifications.connectionError();
            return;
        } else if (req.status == 401) {
            window.notifications.customError("401Error", window.lang.notif("error401Unauthorized"));
        }
        onreadystatechange(req);
    };
    req.send(JSON.stringify(data));
};

export const _download = (url: string, fname: string): void => {
    let req = new XMLHttpRequest();
    if (window.pages) { url = window.pages.Base + url; }
    req.open("GET", url, true);
    req.responseType = 'blob';
    req.setRequestHeader("Authorization", "Bearer " + window.token);
    req.setRequestHeader('Content-Type', 'application/json; charset=UTF-8');
    req.onload = (e: Event) => {
        let link = document.createElement("a") as HTMLAnchorElement;
        link.href = URL.createObjectURL(req.response);
        link.download = fname;
        link.dispatchEvent(new MouseEvent("click"));
    };
    req.send();
};

export const _upload = (url: string, formData: FormData): void => {
    let req = new XMLHttpRequest();
    if (window.pages) { url = window.pages.Base + url; }
    req.open("POST", url, true);
    req.setRequestHeader("Authorization", "Bearer " + window.token);
    // req.setRequestHeader('Content-Type', 'multipart/form-data');
    req.send(formData);
};

export const _req = (method: string, url: string, data: Object, onreadystatechange: (req: XMLHttpRequest) => void, response?: boolean, statusHandler?: (req: XMLHttpRequest) => void, noConnectionError: boolean = false): void => {
    let req = new XMLHttpRequest();
    if (window.pages) { url = window.pages.Base + url; }
    req.open(method, url, true);
    if (response) {
        req.responseType = 'json';
    }
    req.setRequestHeader("Authorization", "Bearer " + window.token);
    req.setRequestHeader('Content-Type', 'application/json; charset=UTF-8');
    req.onreadystatechange = () => {
        if (statusHandler) { statusHandler(req); }
        else if (req.status == 0) {
            if (!noConnectionError) window.notifications.connectionError();
            return;
        } else if (req.status == 401) {
            window.notifications.customError("401Error", window.lang.notif("error401Unauthorized"));
        }
        onreadystatechange(req);
    };
    req.send(JSON.stringify(data));
};

export const _post = (url: string, data: Object, onreadystatechange: (req: XMLHttpRequest) => void, response?: boolean, statusHandler?: (req: XMLHttpRequest) => void, noConnectionError: boolean = false): void => _req("POST", url, data, onreadystatechange, response, statusHandler, noConnectionError);

export const _put = (url: string, data: Object, onreadystatechange: (req: XMLHttpRequest) => void, response?: boolean, statusHandler?: (req: XMLHttpRequest) => void, noConnectionError: boolean = false): void => _req("PUT", url, data, onreadystatechange, response, statusHandler, noConnectionError);

export const _patch = (url: string, data: Object, onreadystatechange: (req: XMLHttpRequest) => void, response?: boolean, statusHandler?: (req: XMLHttpRequest) => void, noConnectionError: boolean = false): void => _req("PATCH", url, data, onreadystatechange, response, statusHandler, noConnectionError);

export function _delete(url: string, data: Object, onreadystatechange: (req: XMLHttpRequest) => void, noConnectionError: boolean = false): void {
    let req = new XMLHttpRequest();
    if (window.pages) { url = window.pages.Base + url; }
    req.open("DELETE", url, true);
    req.setRequestHeader("Authorization", "Bearer " + window.token);
    req.setRequestHeader('Content-Type', 'application/json; charset=UTF-8');
    req.onreadystatechange = () => {
        if (req.status == 0) {
            if (!noConnectionError) window.notifications.connectionError();
            return;
        } else if (req.status == 401) {
            window.notifications.customError("401Error", window.lang.notif("error401Unauthorized"));
        }
        onreadystatechange(req);
    };
    req.send(JSON.stringify(data));
}

export function toClipboard (str: string) {
    const el = document.createElement('textarea') as HTMLTextAreaElement;
    el.value = str;
    el.readOnly = true;
    el.style.position = "absolute";
    el.style.left = "-9999px";
    document.body.appendChild(el);
    const selected = document.getSelection().rangeCount > 0 ? document.getSelection().getRangeAt(0) : false;
    el.select();
    document.execCommand("copy");
    document.body.removeChild(el);
    if (selected) {
        document.getSelection().removeAllRanges();
        document.getSelection().addRange(selected);
    }
}

export class notificationBox implements NotificationBox {
    private _box: HTMLDivElement;
    private _errorTypes: { [type: string]: boolean } = {};
    private _positiveTypes: { [type: string]: boolean } = {};
    timeout: number;
    constructor(box: HTMLDivElement, timeout?: number) { this._box = box; this.timeout = timeout || 5; }

    private _error = (message: string): HTMLElement => {
        const noti = document.createElement('aside');
        noti.classList.add("aside", "~critical", "@low", "mt-2", "notification-error");
        let error = "";
        if (window.lang) {
            error = window.lang.strings("error") + ":"
        }
        noti.innerHTML = `<strong>${error}</strong> ${message}`;
        const closeButton = document.createElement('span') as HTMLSpanElement;
        closeButton.classList.add("button", "~critical", "@low", "ml-4");
        closeButton.innerHTML = `<i class="icon ri-close-line"></i>`;
        closeButton.onclick = () => this._close(noti);
        noti.classList.add("animate-slide-in");
        noti.appendChild(closeButton);
        return noti;
    }
    
    private _positive = (bold: string, message: string): HTMLElement => {
        const noti = document.createElement('aside');
        noti.classList.add("aside", "~positive", "@low", "mt-2", "notification-positive");
        noti.innerHTML = `<strong>${bold}</strong> ${message}`;
        const closeButton = document.createElement('span') as HTMLSpanElement;
        closeButton.classList.add("button", "~positive", "@low", "ml-4");
        closeButton.innerHTML = `<i class="icon ri-close-line"></i>`;
        closeButton.onclick = () => this._close(noti); 
        noti.classList.add("animate-slide-in");
        noti.appendChild(closeButton);
        return noti;
    }
    
    private _close = (noti: HTMLElement) => {
        noti.classList.remove("animate-slide-in");
        noti.classList.add("animate-slide-out");
        noti.addEventListener(window.animationEvent, () => {
            this._box.removeChild(noti); 
        }, false);
    }

    
    connectionError = () => { this.customError("connectionError", window.lang.notif("errorConnection")); }

    customError = (type: string, message: string) => {
        this._errorTypes[type] = this._errorTypes[type] || false;
        const noti = this._error(message);
        noti.classList.add("error-" + type);
        const previousNoti: HTMLElement | undefined = this._box.querySelector("aside.error-" + type);
        if (this._errorTypes[type] && previousNoti !== undefined && previousNoti != null) {
            this._box.removeChild(previousNoti);
            noti.classList.add("animate-pulse");
            noti.classList.remove("animate-slide-in");
        }
        this._box.appendChild(noti);
        this._errorTypes[type] = true;
        setTimeout(() => { if (this._box.contains(noti)) { this._close(noti); this._errorTypes[type] = false; } }, this.timeout*1000);
    }
    
    customPositive = (type: string, bold: string, message: string) => {
        this._positiveTypes[type] = this._positiveTypes[type] || false;
        const noti = this._positive(bold, message);
        noti.classList.add("positive-" + type);
        const previousNoti: HTMLElement | undefined = this._box.querySelector("aside.positive-" + type);
        if (this._positiveTypes[type] && previousNoti !== undefined && previousNoti != null) {
            this._box.removeChild(previousNoti);
            noti.classList.add("animate-pulse");
            noti.classList.remove("animate-slide-in");
        }
        this._box.appendChild(noti);
        this._positiveTypes[type] = true;
        setTimeout(() => { if (this._box.contains(noti)) { this._close(noti); this._positiveTypes[type] = false; } }, this.timeout*1000);
    }

    customSuccess = (type: string, message: string) => this.customPositive(type, window.lang.strings("success") + ":", message)
}

export const whichAnimationEvent = () => {
    const el = document.createElement("fakeElement");
    if (el.style["animation"] !== void 0) {
        return "animationend";
    }
    return "webkitAnimationEnd";
}

export function toggleLoader(el: HTMLElement, small: boolean = true) {
    if (el.classList.contains("loader")) {
        el.classList.remove("loader");
        el.classList.remove("loader-sm");
        const dot = el.querySelector("span.dot");
        if (dot) { dot.remove(); }
    } else {
        el.classList.add("loader");
        if (small) { el.classList.add("loader-sm"); }
        const dot = document.createElement("span") as HTMLSpanElement;
        dot.classList.add("dot")
        el.appendChild(dot);
    }
}

export function addLoader(el: HTMLElement, small: boolean = true, relative: boolean = false) {
    if (el.classList.contains("loader")) return;
    el.classList.add("loader");
    if (relative) el.classList.add("rel");
    if (small) { el.classList.add("loader-sm"); }
    const dot = document.createElement("span") as HTMLSpanElement;
    dot.classList.add("dot")
    el.appendChild(dot);
}

export function removeLoader(el: HTMLElement, small: boolean = true) {
    if (el.classList.contains("loader")) {
        el.classList.remove("loader");
        el.classList.remove("loader-sm");
        el.classList.remove("rel");
        const dot = el.querySelector("span.dot");
        if (dot) { dot.remove(); }
    }
}

export function insertText(textarea: HTMLTextAreaElement, text: string) {
    // https://kubyshkin.name/posts/insert-text-into-textarea-at-cursor-position <3
    const isSuccess = document.execCommand("insertText", false, text);

    // Firefox (non-standard method)
    if (!isSuccess && typeof textarea.setRangeText === "function") {
        const start = textarea.selectionStart;
        textarea.setRangeText(text);
        // update cursor to be at the end of insertion
        textarea.selectionStart = textarea.selectionEnd = start + text.length;

        // Notify any possible listeners of the change
        const e = document.createEvent("UIEvent");
        e.initEvent("input", true, false);
        textarea.dispatchEvent(e);
        textarea.focus();
    }
}

export function bindManualDropdowns() {
    const buttons = Array.from(document.getElementsByClassName("dropdown-manual-toggle") as HTMLCollectionOf<HTMLSpanElement>);
    for (let button of buttons) {
        const parent = button.closest(".dropdown.manual");
        const display = parent.querySelector(".dropdown-display");
        const mousein = () => parent.classList.add("selected");
        const mouseout = () => parent.classList.remove("selected");
        button.addEventListener("mouseover", mousein);
        button.addEventListener("mouseout", mouseout);
        display.addEventListener("mouseover",  mousein);
        display.addEventListener("mouseout", mouseout);
        button.onclick = () => {
            parent.classList.add("selected");
            document.addEventListener("click", outerClickListener);
            button.removeEventListener("mouseout", mouseout);
            display.removeEventListener("mouseout", mouseout);
        };
        const outerClickListener = (event: Event) => {
            if (!(event.target instanceof HTMLElement && (display.contains(event.target) || button.contains(event.target)))) {
                parent.classList.remove("selected");
                document.removeEventListener("click", outerClickListener);
                button.addEventListener("mouseout", mouseout);
                display.addEventListener("mouseout", mouseout);
            }
        };
    }
}

export function unicodeB64Decode(s: string): string {
    const decoded = atob(s);
    const byteArray = Uint8Array.from(decoded, (m) => m.codePointAt(0));
    const toUnicode = new TextDecoder().decode(byteArray);
    return toUnicode;
}

export function unicodeB64Encode(s: string): string {
    const encoded = new TextEncoder().encode(s);
    const bin = String.fromCodePoint(...encoded);
    return btoa(bin);
}

// Only allow running a function every n milliseconds.
// Source: Clément Prévost at https://stackoverflow.com/questions/27078285/simple-throttle-in-javascript
// function foo<T>(bar: T): T {
export function throttle (callback: () => void, limitMilliseconds: number): () => void {
    var waiting = false;                      // Initially, we're not waiting
    return function () {                      // We return a throttled function
        if (!waiting) {                       // If we're not waiting
            callback.apply(this, arguments);  // Execute users function
            waiting = true;                   // Prevent future invocations
            setTimeout(function () {          // After a period of time
                waiting = false;              // And allow future invocations
            }, limitMilliseconds);
        }
    }
}

export function SetupCopyButton(button: HTMLButtonElement, text: string, baseClass?: string, notif?: string) {
    if (!notif) notif = window.lang.strings("copied");
    if (!baseClass) baseClass = "~info";
    // script will probably turn this into multiple
    const baseClasses = baseClass.split(" ");
    button.type = "button";
    button.classList.add("button", ...baseClasses, "@low", "p-1");
    button.title = window.lang.strings("copy");
    const icon = document.createElement("i");
    icon.classList.add("icon", "ri-file-copy-line");
    button.appendChild(icon)
    button.onclick = () => { 
        toClipboard(text);
        icon.classList.remove("ri-file-copy-line");
        icon.classList.add("ri-check-line");
        button.classList.remove(...baseClasses);
        button.classList.add("~positive");
        setTimeout(() => {
            icon.classList.remove("ri-check-line");
            icon.classList.add("ri-file-copy-line");
            button.classList.remove("~positive");
            button.classList.add(...baseClasses);
        }, 800);
        window.notifications.customPositive("copied", "", notif);
    };
}

export function CopyButton(text: string, baseClass?: string, notif?: string): HTMLButtonElement {
    const button = document.createElement("button");
    SetupCopyButton(button, text, baseClass, notif);
    return button;
}
