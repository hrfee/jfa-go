import {addLoader, removeLoader, _get} from "../modules/common.js";

export interface DiscordUser {
    name: string;
    avatar_url: string;
    id: string;
}

var listeners: { [buttonText: string]: (event: CustomEvent) => void } = {};

export function newDiscordSearch(title: string, description: string, buttonText: string, buttonFunction: (user: DiscordUser, passData: string) => void): (passData: string) => void {
    if (!window.discordEnabled) {
        return () => {};
    }
    let timer: NodeJS.Timer;
    listeners[buttonText] = (event: CustomEvent) => {
        clearTimeout(timer);
        const list = document.getElementById("discord-list") as HTMLTableElement;
        const input = document.getElementById("discord-search") as HTMLInputElement;
        if (input.value.length < 2) {
            return;
        }
        list.innerHTML = ``;
        addLoader(list);
        list.parentElement.classList.add("mb-4", "mt-4");
        timer = setTimeout(() => {
            _get("/users/discord/" + input.value, null, (req: XMLHttpRequest) => {
                if (req.readyState == 4) {
                    if (req.status != 200) {
                        removeLoader(list);
                        list.parentElement.classList.remove("mb-4", "mt-4");
                        return;
                    }
                    const users = req.response["users"] as Array<DiscordUser>;
                    let innerHTML = ``;
                    for (let i = 0; i < users.length; i++) {
                        innerHTML += `
                        <tr>
                            <td class="img-circle sm">
                                <img class="img-circle" src="${users[i].avatar_url}" width="32" height="32">
                            </td>
                            <td class="sm">
                                <p class="content">${users[i].name}</p>
                            </td>
                            <td class="sm float-right">
                                <span id="discord-user-${users[i].id}" class="button ~info @high">${buttonText}</span>
                            </td>
                        </tr>
                        `;
                    }
                    list.innerHTML = innerHTML;
                    removeLoader(list);
                    list.parentElement.classList.remove("mb-4", "mt-4");
                    for (let i = 0; i < users.length; i++) {
                        const button = document.getElementById(`discord-user-${users[i].id}`) as HTMLInputElement;
                        button.onclick = () => buttonFunction(users[i], event.detail);
                    }
                }
            });
        }, 750);
    }

    return (passData: string) => {
        const input = document.getElementById("discord-search") as HTMLInputElement;
        const list = document.getElementById("discord-list") as HTMLDivElement;
        const header = document.getElementById("discord-header") as HTMLSpanElement;
        const desc = document.getElementById("discord-description") as HTMLParagraphElement;
        desc.textContent = description;
        header.textContent = title;
        list.innerHTML = ``;
        input.value = "";
        for (let key in listeners) {
            input.removeEventListener("keyup", listeners[key]);
        }
        input.addEventListener("keyup", listeners[buttonText].bind(null, { detail: passData }));

        window.modals.discord.show();
    }
}
