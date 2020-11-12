declare var window: Window;

export function serializeForm(id: string): Object {
    const form = document.getElementById(id) as HTMLFormElement;
    let formData = {};
    for (let i = 0; i < form.elements.length; i++) {
        const el = form.elements[i];
        if ((el as HTMLInputElement).type == "submit") {
            continue;
        }
        let name = (el as HTMLInputElement).name;
        if (!name) {
            name = el.id;
        }
        switch ((el as HTMLInputElement).type) {
            case "checkbox":
                formData[name] = (el as HTMLInputElement).checked;
                break;
            case "text":
            case "password":
            case "email":
            case "number":
                formData[name] = (el as HTMLInputElement).value;
                break;
            case "select-one":
            case "select":
                let val: string = (el as HTMLSelectElement).value.toString();
                if (!isNaN(val as any)) {
                    formData[name] = +val;
                } else {
                    formData[name] = val;
                }
                break;
        }
    }
    return formData;
}

export const rmAttr = (el: HTMLElement, attr: string): void => {
    if (el.classList.contains(attr)) {
        el.classList.remove(attr);
    }
};

export const addAttr = (el: HTMLElement, attr: string): void => el.classList.add(attr);

export const _get = (url: string, data: Object, onreadystatechange: () => void): void => {
    let req = new XMLHttpRequest();
    req.open("GET", url, true);
    req.responseType = 'json';
    req.setRequestHeader("Authorization", "Bearer " + btoa(window.token));
    req.setRequestHeader('Content-Type', 'application/json; charset=UTF-8');
    req.onreadystatechange = onreadystatechange;
    req.send(JSON.stringify(data));
};

export const _post = (url: string, data: Object, onreadystatechange: () => void, response?: boolean): void => {
    let req = new XMLHttpRequest();
    req.open("POST", url, true);
    if (response) {
        req.responseType = 'json';
    }
    req.setRequestHeader("Authorization", "Bearer " + btoa(window.token));
    req.setRequestHeader('Content-Type', 'application/json; charset=UTF-8');
    req.onreadystatechange = onreadystatechange;
    req.send(JSON.stringify(data));
};

export function _delete(url: string, data: Object, onreadystatechange: () => void): void {
    let req = new XMLHttpRequest();
    req.open("DELETE", url, true);
    req.setRequestHeader("Authorization", "Bearer " + btoa(window.token));
    req.setRequestHeader('Content-Type', 'application/json; charset=UTF-8');
    req.onreadystatechange = onreadystatechange;
    req.send(JSON.stringify(data));
}

