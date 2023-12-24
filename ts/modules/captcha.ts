import { _get, _post } from "./common.js";

export class Captcha {
    isPWR = false;
    enabled = true;
    verified = false;
    captchaID = "";
    input = document.getElementById("captcha-input") as HTMLInputElement;
    checkbox = document.getElementById("captcha-success") as HTMLSpanElement;
    previous = "";
    reCAPTCHA = false;
    code = "";

    get value(): string { return this.input.value; }

    hasChanged = (): boolean => { return this.value != this.previous; }

    baseValidatorWrapper = (_baseValidator: (oncomplete: (valid: boolean) => void, captchaValid: boolean) => void) => {
        return (oncomplete: (valid: boolean) => void): void => {
            if (this.enabled && !this.reCAPTCHA && this.hasChanged()) {
                this.previous = this.value;
                this.verify(() => {
                    _baseValidator(oncomplete, this.verified);
                });
            } else {
                _baseValidator(oncomplete, this.verified);
            }
        };
    };

    verify = (callback: () => void) => _post("/captcha/verify/" + this.code + "/" + this.captchaID + "/" + this.input.value + (this.isPWR ? "?pwr=true" : ""), null, (req: XMLHttpRequest) => {
        if (req.readyState == 4) {
            if (req.status == 204) {
                this.checkbox.innerHTML = `<i class="ri-check-line"></i>`;
                this.checkbox.classList.add("~positive");
                this.checkbox.classList.remove("~critical");
                this.verified = true;
            } else {
                this.checkbox.innerHTML = `<i class="ri-close-line"></i>`;
                this.checkbox.classList.add("~critical");
                this.checkbox.classList.remove("~positive");
                this.verified = false;
            }
            callback();
        }
    });
    
    generate = () => _get("/captcha/gen/"+this.code+(this.isPWR ? "?pwr=true" : ""), null, (req: XMLHttpRequest) => {
        if (req.readyState == 4) {
            if (req.status == 200) {
                this.captchaID = this.isPWR ? this.code : req.response["id"];
                // the Math.random() appearance below is used for PWRs, since they don't have a unique captchaID. The parameter is ignored by the server, but tells the browser to reload the image.
                document.getElementById("captcha-img").innerHTML = `
                <img class="w-full" src="${window.location.toString().substring(0, window.location.toString().lastIndexOf("/invite"))}/captcha/img/${this.code}/${this.isPWR ? Math.random() : this.captchaID}${this.isPWR ? "?pwr=true" : ""}"></img>
                `;
                this.input.value = "";
            }
        }
    });

    constructor(code: string, enabled: boolean, reCAPTCHA: boolean, isPWR: boolean) {
        this.code = code;
        this.enabled = enabled;
        this.reCAPTCHA = reCAPTCHA;
        this.isPWR = isPWR;
    }
}

export interface GreCAPTCHA {
    render: (container: HTMLDivElement, parameters: {
        sitekey?: string,
        theme?: string,
        size?: string,
        tabindex?: number,
        "callback"?: () => void,
        "expired-callback"?: () => void,
        "error-callback"?: () => void
    }) => void;
    getResponse: (opt_widget_id?: HTMLDivElement) => string;
}

