import { _get, _post, toggleLoader, toDateString } from "../modules/common.js";
import { Marked, Renderer } from "@ts-stack/markdown";

interface updateDTO {
    new: boolean;
    update: Update;
}

export class Updater implements updater {
    private _update: Update;
    private _date: number;
    updateAvailable = false;

    checkForUpdates = (run?: (req: XMLHttpRequest) => void) => _get("/config/update", null, (req: XMLHttpRequest) => {
        if (req.readyState == 4) {
            if (req.status != 200) {
                window.notifications.customError("errorCheckUpdate", window.lang.notif("errorCheckUpdate"));
                return
            }
            let resp = req.response as updateDTO;
            if (resp.new) {
                this.update = resp.update;
                if (run) { run(req); }
            // } else {
            //     window.notifications.customPositive("noUpdatesAvailable", "", window.lang.notif("noUpdatesAvailable"));
            }
        }
    });
    get date(): number { return this._date; }
    set date(unix: number) {
        this._date = unix;
        document.getElementById("update-date").textContent = toDateString(new Date(this._date * 1000));
    }
    
    get description(): string { return this._update.description; }
    set description(description: string) {
        this._update.description = description;
        const el = document.getElementById("update-description") as HTMLParagraphElement;
        el.textContent = description;
        if (this.version == "git") {
            el.classList.add("font-mono", "bg-inherit");
        } else {
            el.classList.remove("font-mono", "bg-inherit");
        }
    }

    get changelog(): string { return this._update.changelog; }
    set changelog(changelog: string) {
        this._update.changelog = changelog;

        document.getElementById("update-changelog").innerHTML = Marked.parse(changelog);
    }

    get version(): string { return this._update.version; }
    set version(version: string) {
        this._update.version = version;
        document.getElementById("update-version").textContent = version;
    }

    get commit(): string { return this._update.commit; }
    set commit(commit: string) {
        this._update.commit = commit;
        document.getElementById("update-commit").textContent = commit.slice(0, 7);
    }

    get link(): string { return this._update.link; }
    set link(link: string) {
        this._update.link = link;
        (document.getElementById("update-version") as HTMLAnchorElement).href = link;
    }

    get download_link(): string { return this._update.download_link; }
    set download_link(link: string) { this._update.download_link = link; }

    get can_update(): boolean { return this._update.can_update; }
    set can_update(can: boolean) {
        this._update.can_update = can;
        const download = document.getElementById("update-download") as HTMLSpanElement;
        const update = document.getElementById("update-update") as HTMLSpanElement;
        if (can) {
            download.classList.add("unfocused");
            update.classList.remove("unfocused");
        } else {
            download.onclick = () => window.open(this._update.download_link || this._update.link);
            download.classList.remove("unfocused");
            update.classList.add("unfocused");
        }
    }

    get update(): Update { return this._update; }
    set update(update: Update) {
        this._update = update;
        this.version = update.version;
        this.commit = update.commit;
        this.date = update.date;
        this.description = update.description;
        this.changelog = update.changelog;
        this.link = update.link;
        this.download_link = update.download_link;
        this.can_update = update.can_update;
    }

    constructor() {
        const update = document.getElementById("update-update") as HTMLSpanElement;
        update.onclick = () => {
            toggleLoader(update);
            _post("/config/update", null, (req: XMLHttpRequest) => {
                if (req.readyState == 4) {
                    toggleLoader(update);
                    const success = req.response["success"] as Boolean;
                    if (req.status == 500 && success) {
                        window.notifications.customSuccess("applyUpdate", window.lang.notif("updateAppliedRefresh"));
                    } else if (req.status != 200) {
                        window.notifications.customError("applyUpdateError", window.lang.notif("errorApplyUpdate"));
                    } else {
                        window.notifications.customSuccess("applyUpdate", window.lang.notif("updateAppliedRefresh"));
                    }
                    window.modals.updateInfo.close();
                }
            }, true, (req: XMLHttpRequest) => {
                 if (req.status == 0) {
                    window.notifications.customSuccess("applyUpdate", window.lang.notif("updateAppliedRefresh"));
                 }
            });
        };
        this.checkForUpdates(() => {
            this.updateAvailable = true;
            window.notifications.customPositive("updateAvailable", "", window.lang.notif("updateAvailable"));
        });
    }
}
