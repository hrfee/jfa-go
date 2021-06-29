import { Modal } from "../../ts/modules/modal.js";
import { whichAnimationEvent } from "../../ts/modules/common.js";

interface window extends Window {
    animationEvent: string;
}

declare var window: window;

window.animationEvent = whichAnimationEvent();

const debModal = new Modal(document.getElementById("modal-deb"));
const debButton = document.getElementById("download-deb") as HTMLAnchorElement;
debButton.onclick = debModal.toggle;

const debUnstableModal = new Modal(document.getElementById("modal-deb-unstable"));
const debUnstableButton = document.getElementById("download-deb-unstable") as HTMLAnchorElement;
debUnstableButton.onclick = debUnstableModal.toggle;

const stableSect = document.getElementById("sect-stable");
const unstableSect = document.getElementById("sect-unstable");

const stableButton = document.getElementById("download-stable") as HTMLSpanElement;
const unstableButton = document.getElementById("download-unstable") as HTMLSpanElement;

stableButton.onclick = () => {
    stableButton.classList.add("!high");
    unstableButton.classList.remove("!high");
    stableSect.classList.remove("unfocused");
    unstableSect.classList.add("unfocused");
}

unstableButton.onclick = () => {
    unstableButton.classList.add("!high");
    stableButton.classList.remove("!high");
    stableSect.classList.add("unfocused");
    unstableSect.classList.remove("unfocused");
}
