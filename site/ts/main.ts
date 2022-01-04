import { Modal } from "../../ts/modules/modal.js";
import { whichAnimationEvent } from "../../ts/modules/common.js";
import { loadBuilds } from "./repo.js";

interface window extends Window {
    animationEvent: string;
}

declare var window: window;

window.animationEvent = whichAnimationEvent();

const debModal = new Modal(document.getElementById("modal-deb"));
const debButton = document.getElementById("download-deb") as HTMLAnchorElement;
debButton.onclick = debModal.toggle;

const debUnstable = document.getElementById("deb-unstable");
const debUnstableButton = document.getElementById("download-deb-unstable") as HTMLAnchorElement;
debUnstableButton.onclick = debModal.toggle;

const stableSect = document.getElementById("sect-stable");
export const unstableSect = document.getElementById("sect-unstable");

const stableButton = document.getElementById("download-stable") as HTMLSpanElement;
const unstableButton = document.getElementById("download-unstable") as HTMLSpanElement;

const dockerUnstable = document.getElementById("docker-unstable");

stableButton.onclick = () => {
    debUnstable.classList.add("unfocused");
    dockerUnstable.classList.add("unfocused");
    stableButton.classList.add("@high");
    stableButton.classList.remove("@low");
    unstableButton.classList.remove("@high");
    stableSect.classList.remove("unfocused");
    unstableSect.classList.add("unfocused");

}

unstableButton.onclick = () => {
    debUnstable.classList.remove("unfocused");
    dockerUnstable.classList.remove("unfocused");
    unstableButton.classList.add("@high");
    unstableButton.classList.remove("@low");
    stableButton.classList.remove("@high");
    stableSect.classList.add("unfocused");
    unstableSect.classList.remove("unfocused");
}

const dockerModal = new Modal(document.getElementById("modal-docker"));
const dockerButton = document.getElementById("download-docker") as HTMLSpanElement;
const dockerUnstableButton = document.getElementById("download-docker-unstable") as HTMLSpanElement;

dockerButton.onclick = dockerModal.toggle;
dockerUnstableButton.onclick = dockerModal.toggle;

loadBuilds();
