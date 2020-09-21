function serializeForm(id: string): Object {
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
                let val: string | number = (el as HTMLSelectElement).value;
                if (+val != NaN) {
                    val = +val;
                }
                formData[name] = val;
                break;
        }
    }
    return formData;
}
