const buttonNormal = document.getElementById("button-log-normal") as HTMLInputElement;
const buttonSanitized = document.getElementById("button-log-sanitized") as HTMLInputElement;

const logNormal = document.getElementById("log-normal") as HTMLInputElement;
const logSanitized = document.getElementById("log-sanitized") as HTMLInputElement;

const buttonChange = (type: string) => {
    console.log("RUN");
    if (type == "normal") {
        logSanitized.classList.add("unfocused");
        logNormal.classList.remove("unfocused");
        buttonNormal.classList.add("!high");
        buttonNormal.classList.remove("!normal");
        buttonSanitized.classList.add("!normal");
        buttonSanitized.classList.remove("!high");
    } else {
        logNormal.classList.add("unfocused");
        logSanitized.classList.remove("unfocused");
        buttonSanitized.classList.add("!high");
        buttonSanitized.classList.remove("!normal");
        buttonNormal.classList.add("!normal");
        buttonNormal.classList.remove("!high");
    }
}
buttonNormal.onclick = () => buttonChange("normal");
buttonSanitized.onclick = () => buttonChange("sanitized");
