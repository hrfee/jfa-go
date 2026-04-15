export abstract class TableRow {
    protected _row: HTMLTableRowElement;

    remove() {
        this._row.remove();
    }
    asElement(): HTMLTableRowElement {
        return this._row;
    }

    constructor() {
        this._row = document.createElement("tr");
        this._row.classList.add("border-b", "border-dashed", "dark:border-dotted", "dark:border-stone-700");
    }
}
