package main

type langMeta struct {
	Name string `json:"name"`
}

type quantityString struct {
	Singular string `json:"singular"`
	Plural   string `json:"plural"`
}

type adminLang struct {
	Meta            langMeta                  `json:"meta"`
	Strings         map[string]string         `json:"strings"`
	Notifications   map[string]string         `json:"notifications"`
	QuantityStrings map[string]quantityString `json:"quantityStrings"`
}

type formLang struct {
	Strings           map[string]string         `json:"strings"`
	ValidationStrings map[string]quantityString `json:"validationStrings"`
}
