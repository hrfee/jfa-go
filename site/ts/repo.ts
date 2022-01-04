import { _get, _post, _delete } from "../../ts/modules/common.js";
import { unstableSect } from "./main.js";

const urlBase = "https://builds.hrfee.pw/repo/hrfee/jfa-go/latest/file/";
const categories = {
    "Windows (Tray)": {
        "x64": "Windows"
    },
    "Linux (Tray)": {
        "x64": "TrayIcon_Linux_x86_64.zip"
    },
    "Linux": {
        "x64": "Linux_x86_64.zip",
        "ARM (32-bit)": "Linux_arm.zip",
        "ARM (64-bit)": "Linux_arm64.zip"
    },
    "macOS": {
        "x64": "macOS_x86_64",
        "ARM": "macOS_arm64"
    }
};

export const loadBuilds = () => {
    for (let buildName in categories) {
        if (Object.keys(categories[buildName]).length == 1) {
            const button = document.createElement("a") as HTMLAnchorElement;
            button.classList.add("button", "~info", "mr-2", "mb-2", "lang-link");
            button.target = "_blank";
            button.textContent = buildName.toLowerCase();
            button.href = urlBase + categories[buildName][Object.keys(categories[buildName])[0]];
            unstableSect.querySelector(".row.col.flex.center").appendChild(button);
        } else {
            const dropdown = document.createElement("span") as HTMLSpanElement;
            dropdown.tabIndex = 0;
            dropdown.classList.add("dropdown");
            let innerHTML = `
            <span class="button ~info mr-2 mb-2 lang-link">
                ${buildName.toLowerCase()}
                <span class="ml-2 chev"></span>
            </span>
            <div class="dropdown-display above">
                <div class="card @low">
            `;
            for (let arch in categories[buildName]) {
                innerHTML += `
                <a href="${urlBase + categories[buildName][arch]}" target="_blank" class="button input ~neutral field mb-2 lang-link">${arch}</a>
                `;
            }
            innerHTML += `
                </div>
            </div>
            `;
            dropdown.innerHTML = innerHTML;
            unstableSect.querySelector(".row.col.flex.center").appendChild(dropdown);
        }
    }
};
