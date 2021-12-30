import { toClipboard } from "./modules/common.js";

const buttonNormal = document.getElementById("button-log-normal") as HTMLInputElement;
const buttonSanitized = document.getElementById("button-log-sanitized") as HTMLInputElement;

const logNormal = document.getElementById("log-normal") as HTMLInputElement;
const logSanitized = document.getElementById("log-sanitized") as HTMLInputElement;

const buttonChange = (type: string) => {
    console.log("RUN");
    if (type == "normal") {
        logSanitized.classList.add("unfocused");
        logNormal.classList.remove("unfocused");
        buttonNormal.classList.add("@high");
        buttonNormal.classList.remove("@low");
        buttonSanitized.classList.add("@low");
        buttonSanitized.classList.remove("@high");
    } else {
        logNormal.classList.add("unfocused");
        logSanitized.classList.remove("unfocused");
        buttonSanitized.classList.add("@high");
        buttonSanitized.classList.remove("@low");
        buttonNormal.classList.add("@low");
        buttonNormal.classList.remove("@high");
    }
}
buttonNormal.onclick = () => buttonChange("normal");
buttonSanitized.onclick = () => buttonChange("sanitized");

const copyButton = document.getElementById("copy-log") as HTMLSpanElement;
copyButton.onclick = () => {
    if (logSanitized.classList.contains("unfocused")) {
        toClipboard("```\n" + logNormal.textContent + "```");
    } else {
        toClipboard("```\n" + logSanitized.textContent + "```");
    }
    copyButton.textContent = "Copied.";
    setTimeout(() => { copyButton.textContent = "Copy"; }, 1500);
};
