function serializeForm(id) {
    var form = document.getElementById(id);
    var formData = {};
    for (var i = 0; i < form.elements.length; i++) {
        var el = form.elements[i];
        if (el.type != 'submit') {
            var name = el.name;
            if (name == '') {
                name = el.id;
            };
            switch (el.type) {
                case 'checkbox':
                    formData[name] = el.checked;
                    break;
                case 'text':
                case 'password':
                case 'email':
                case 'number':
                    formData[name] = el.value;
                    break;
                case 'select-one':
                    let val = el.value;
                    if (!isNaN(val)) {
                        val = parseInt(val)
                    }
                    formData[name] = val;
                    break;
            };
        };
    };
    return formData;
};
