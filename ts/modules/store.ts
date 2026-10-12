import { _get, _post, addLoader, removeLoader, toDateString } from "./common.js";

declare var window: GlobalWindow;

interface PaymentRecord {
    id: string;
    provider: string;
    instance_id?: string;
    provider_payment_id: string;
    provider_live_mode?: boolean;
    customer_id?: string;
    payment_intent_id?: string;
    charge_id?: string;
    invoice_id?: string;
    subscription_id?: string;
    target_email: string;
    plan_id?: string;
    plan: string;
    profile: string;
    access_months?: number;
    access_days?: number;
    recurring?: boolean;
    stripe_interval?: string;
    stripe_interval_count?: number;
    amount: number;
    refunded_amount?: number;
    currency: string;
    status: string;
    email_status: string;
    invoice_status?: string;
    subscription_status?: string;
    subscription_cancel_at?: number;
    subscription_cancel_at_period_end?: boolean;
    subscription_canceled_at?: number;
    subscription_ended_at?: number;
    invite_code?: string;
    jellyfin_id?: string;
    error?: string;
    created: number;
    updated: number;
    paid_at?: number;
    fulfilled_at?: number;
    email_sent_at?: number;
    last_reconciled_at?: number;
}

interface PaymentsResponse {
    payments: PaymentRecord[];
}

interface PaymentPlan {
    id: string;
    name: string;
    description?: string;
    enabled: boolean;
    price: number;
    currency: string;
    recurring: boolean;
    stripe_interval?: string;
    stripe_interval_count?: number;
    access_months?: number;
    access_days?: number;
    profile?: string;
}

interface PaymentPlansResponse {
    plans: PaymentPlan[];
}

interface ReconcileResponse {
    scanned: number;
    matched: number;
    created: number;
    updated: number;
    skipped: number;
    refreshed: number;
    lifecycle_updates: number;
    needs_review: number;
    error?: string;
}

interface CancelSubscriptionResponse {
    payment: PaymentRecord;
    subscription_id: string;
    refund_id?: string;
}

export class Payments {
    private _ledgerButton: HTMLButtonElement;
    private _storeButton: HTMLButtonElement;
    private _ledgerPanel: HTMLElement;
    private _storePanel: HTMLElement;
    private _planList: HTMLElement;
    private _planEditor: HTMLElement;
    private _planEditorEmpty: HTMLElement;
    private _addPlanButton: HTMLButtonElement;
    private _saveButton: HTMLButtonElement;
    private _refreshButton: HTMLButtonElement;
    private _reconcileButton: HTMLButtonElement;
    private _reconcileStatus: HTMLElement;
    private _list: HTMLTableSectionElement;
    private _empty: HTMLElement;
    private _cancelForm: HTMLFormElement;
    private _cancelSummary: HTMLElement;
    private _cancelCustom: HTMLInputElement;
    private _cancelRefund: HTMLInputElement;
    private _cancelSubmit: HTMLSpanElement;
    private _cancelPayment: PaymentRecord = null;
    private _plans: PaymentPlan[] = [];
    private _selectedPlanIndex = 0;

    constructor() {
        this._ledgerButton = document.getElementById("payments-view-ledger") as HTMLButtonElement;
        this._storeButton = document.getElementById("payments-view-store") as HTMLButtonElement;
        this._ledgerPanel = document.getElementById("payments-view-ledger-panel");
        this._storePanel = document.getElementById("payments-view-store-panel");
        this._planList = document.getElementById("payments-plan-list") as HTMLElement;
        this._planEditor = document.getElementById("payments-plan-editor") as HTMLElement;
        this._planEditorEmpty = document.getElementById("payments-plan-editor-empty") as HTMLElement;
        this._addPlanButton = document.getElementById("payments-plan-add") as HTMLButtonElement;
        this._saveButton = document.getElementById("payments-save") as HTMLButtonElement;
        this._refreshButton = document.getElementById("payments-refresh") as HTMLButtonElement;
        this._reconcileButton = document.getElementById("payments-reconcile") as HTMLButtonElement;
        this._reconcileStatus = document.getElementById("payments-reconcile-status");
        this._list = document.getElementById("payments-list") as HTMLTableSectionElement;
        this._empty = document.getElementById("payments-empty");
        this._cancelForm = document.getElementById("form-payment-subscription-cancel") as HTMLFormElement;
        this._cancelSummary = document.getElementById("payment-subscription-cancel-summary");
        this._cancelCustom = document.getElementById("payment-subscription-cancel-custom") as HTMLInputElement;
        this._cancelRefund = document.getElementById("payment-subscription-cancel-refund") as HTMLInputElement;
        this._cancelSubmit = document.getElementById("payment-subscription-cancel-submit") as HTMLSpanElement;

        this._ledgerButton.onclick = () => this.showSection("ledger");
        this._storeButton.onclick = () => this.showSection("store");
        this._addPlanButton.onclick = this.addPlan;
        this._saveButton.onclick = this.save;
        this._refreshButton.onclick = this.loadPayments;
        this._reconcileButton.onclick = this.reconcileStripe;
        this._cancelForm.onsubmit = this.cancelSubscription;
        this.showSection("ledger");
    }

    load = () => {
        this.loadPlans();
        this.loadPayments();
    };

    private showSection = (section: "ledger" | "store") => {
        const ledger = section == "ledger";
        this._ledgerPanel.classList.toggle("unfocused", !ledger);
        this._storePanel.classList.toggle("unfocused", ledger);
        this.setSectionButton(this._ledgerButton, ledger);
        this.setSectionButton(this._storeButton, !ledger);
    };

    private setSectionButton = (button: HTMLButtonElement, active: boolean) => {
        button.classList.toggle("~urge", active);
        button.classList.toggle("@high", active);
        button.classList.toggle("~neutral", !active);
        button.classList.toggle("@low", !active);
        button.setAttribute("aria-pressed", active ? "true" : "false");
    };

    private loadPlans = () => {
        _get("/payments/plans", null, (req: XMLHttpRequest) => {
            if (req.readyState != 4) return;
            if (req.status != 200) {
                window.notifications.customError("paymentPlansLoadError", "Failed to load payment plans");
                return;
            }
            const data = req.response as PaymentPlansResponse;
            this._plans = data.plans || [];
            this._selectedPlanIndex = 0;
            this.renderPlans();
        });
    };

    private loadPayments = () => {
        addLoader(this._refreshButton);
        _get("/payments/list", null, (req: XMLHttpRequest) => {
            if (req.readyState != 4) return;
            removeLoader(this._refreshButton);
            if (req.status != 200) {
                window.notifications.customError("paymentsLoadError", "Failed to load payments");
                return;
            }
            const data = req.response as PaymentsResponse;
            this.renderPayments(data.payments || []);
        });
    };

    private reconcileStripe = () => {
        addLoader(this._reconcileButton);
        this._reconcileStatus.textContent = "";
        _post("/payments/reconcile/stripe", null, (req: XMLHttpRequest) => {
            if (req.readyState != 4) return;
            removeLoader(this._reconcileButton);
            if (req.status != 200) {
                window.notifications.customError("paymentsReconcileError", "Failed to reconcile Stripe payments");
                return;
            }

            const result = req.response as ReconcileResponse;
            this._reconcileStatus.textContent = `Scanned ${result.scanned}, matched ${result.matched}, refreshed ${result.refreshed || 0}, updated ${result.updated}, lifecycle updates ${result.lifecycle_updates || 0}, needs review ${result.needs_review}`;
            window.notifications.customSuccess("paymentsReconciled", "Stripe reconciliation complete");
            this.loadPayments();
        }, true);
    };

    save = () => {
        let plans: PaymentPlan[];
        try {
            plans = this.collectPlans();
        } catch (error) {
            window.notifications.customError("paymentPlansInvalid", error.message || "Payment plans are invalid");
            return;
        }
        const payload = {
            plans: plans,
        };

        addLoader(this._saveButton);
        _post("/payments/plans", payload, (req: XMLHttpRequest) => {
            if (req.readyState != 4) return;
            removeLoader(this._saveButton);
            if (req.status == 200) {
                const data = req.response as PaymentPlansResponse;
                this._plans = data.plans || plans;
                this._selectedPlanIndex = Math.min(this._selectedPlanIndex, this._plans.length - 1);
                this.renderPlans();
                window.notifications.customSuccess("paymentPlansSaved", "Payment plans saved");
            } else {
                window.notifications.customError("paymentPlansSaved", req.response?.error || "Failed to save payment plans");
            }
        }, true);
    };

    private addPlan = () => {
        const id = this.uniquePlanID("new_plan");
        this._plans.push({
            id: id,
            name: "New Plan",
            description: "",
            enabled: false,
            price: 200,
            currency: "usd",
            recurring: true,
            stripe_interval: "month",
            stripe_interval_count: 1,
            access_months: 1,
            access_days: 0,
            profile: "Default",
        });
        this._selectedPlanIndex = this._plans.length - 1;
        this.renderPlans();
    };

    private renderPlans = () => {
        if (this._selectedPlanIndex < 0) this._selectedPlanIndex = 0;
        if (this._selectedPlanIndex >= this._plans.length) this._selectedPlanIndex = Math.max(0, this._plans.length - 1);
        this.renderPlanList();
        this.renderPlanEditor();
    };

    private renderPlanList = () => {
        this._planList.replaceChildren();
        if (this._plans.length == 0) {
            const empty = document.createElement("aside");
            empty.className = "aside sm ~neutral @low";
            empty.textContent = "No plans yet. Add one to create a store option.";
            this._planList.appendChild(empty);
            return;
        }
        for (let i = 0; i < this._plans.length; i++) {
            this._planList.appendChild(this.planButton(this._plans[i], i));
        }
    };

    private planButton = (plan: PaymentPlan, index: number): HTMLButtonElement => {
        const button = document.createElement("button");
        button.type = "button";
        button.className = `button ~neutral ${index == this._selectedPlanIndex ? "@high" : "@low"} flex flex-col gap-1 items-stretch text-left p-3`;
        button.onclick = () => {
            this._selectedPlanIndex = index;
            this.renderPlans();
        };

        const top = document.createElement("div");
        top.className = "flex flex-row gap-2 items-center justify-between";

        const name = document.createElement("span");
        name.className = "font-medium truncate";
        name.textContent = plan.name || "Unnamed plan";
        top.appendChild(name);

        const status = this.badge(plan.enabled ? "Live" : "Hidden", plan.enabled ? "~positive" : "~neutral");
        top.appendChild(status);
        button.appendChild(top);

        const summary = document.createElement("span");
        summary.className = "support text-left";
        summary.textContent = `${this.formatMoney(plan.price, plan.currency)} · ${this.formatPlanBilling(plan)} · ${this.formatPlanAccess(plan)}`;
        button.appendChild(summary);

        const id = document.createElement("span");
        id.className = "support font-mono text-left truncate";
        id.textContent = plan.id || "no-id";
        button.appendChild(id);

        return button;
    };

    private renderPlanEditor = () => {
        const plan = this._plans[this._selectedPlanIndex];
        this._planEditor.replaceChildren();
        this._planEditor.classList.toggle("unfocused", !plan);
        this._planEditorEmpty.classList.toggle("unfocused", Boolean(plan));
        if (!plan) return;

        const header = document.createElement("div");
        header.className = "flex flex-col md:flex-row gap-2 md:items-start md:justify-between";
        const title = document.createElement("div");
        title.className = "flex flex-col gap-1 min-w-0";
        const name = document.createElement("span");
        name.className = "subheading truncate";
        name.textContent = plan.name || "Unnamed plan";
        const summary = document.createElement("span");
        summary.className = "support";
        summary.textContent = `${this.formatMoney(plan.price, plan.currency)} · ${this.formatPlanBilling(plan)} · ${this.formatPlanAccess(plan)}`;
        title.append(name, summary);
        header.appendChild(title);
        const remove = document.createElement("button");
        remove.className = "button ~critical @low gap-1 self-start";
        remove.title = "Delete plan";
        remove.ariaLabel = "Delete plan";
        remove.innerHTML = `<i class="ri-delete-bin-line"></i><span>Delete</span>`;
        remove.onclick = this.deleteSelectedPlan;
        header.appendChild(remove);
        this._planEditor.appendChild(header);

        const preview = document.createElement("div");
        preview.className = "grid grid-cols-1 md:grid-cols-3 gap-2";
        preview.appendChild(this.planStat("Price", this.formatMoney(plan.price, plan.currency), "ri-price-tag-3-line"));
        preview.appendChild(this.planStat("Billing", this.formatPlanBilling(plan), "ri-bank-card-line"));
        preview.appendChild(this.planStat("Access", this.formatPlanAccess(plan), "ri-time-line"));
        this._planEditor.appendChild(preview);

        const storefront = this.editorSection("Storefront", "What buyers see before Stripe.");
        const enabled = this.switchControl("Show on store", plan.enabled, (checked) => {
            plan.enabled = checked;
            this.renderPlanList();
        });
        storefront.appendChild(this.editorField("Visibility", "Hidden plans stay saved, but buyers cannot select them on the public store page.", enabled));
        const nameInput = this.input("payments-plan-name", plan.name || "", "Monthly");
        nameInput.oninput = () => {
            plan.name = nameInput.value;
            this.renderPlanList();
            name.textContent = plan.name || "Unnamed plan";
        };
        storefront.appendChild(this.editorField("Name", "The public label shown on the store and in payment records.", nameInput));
        const priceRow = document.createElement("div");
        priceRow.className = "grid grid-cols-1 sm:grid-cols-[1fr_8rem] gap-2";
        const amount = this.input("payments-plan-price", this.priceInputValue(plan.price), "0.00");
        amount.type = "number";
        amount.step = "0.01";
        amount.min = "0.01";
        amount.oninput = () => {
            plan.price = Math.round((parseFloat(amount.value) || 0) * 100);
            this.renderPlanList();
        };
        priceRow.appendChild(amount);
        priceRow.appendChild(this.currencySelect(plan.currency || "usd", (value) => {
            plan.currency = value;
            this.renderPlanList();
        }));
        storefront.appendChild(this.editorField("Price", "The amount Stripe charges, shown here in normal dollars instead of cents.", priceRow));
        const description = document.createElement("textarea");
        description.className = "textarea full-width ~neutral @low payments-plan-description";
        description.rows = 2;
        description.placeholder = "Short store description";
        description.value = plan.description || "";
        description.oninput = () => {
            plan.description = description.value;
        };
        storefront.appendChild(this.editorField("Description", "Optional public copy shown under the plan name on the store page.", description));
        this._planEditor.appendChild(storefront);

        const access = this.editorSection("Access", "What jfa-go grants after payment.");
        const accessRow = document.createElement("div");
        accessRow.className = "grid grid-cols-1 sm:grid-cols-2 gap-2";
        const months = this.input("payments-plan-access-months", String(plan.access_months || 0), "0");
        months.type = "number";
        months.min = "0";
        months.oninput = () => {
            plan.access_months = parseInt(months.value || "0") || 0;
            this.renderPlanList();
        };
        accessRow.appendChild(this.editorField("Months", "Whole calendar months added to the invite or existing paid user.", months));
        const days = this.input("payments-plan-access-days", String(plan.access_days || 0), "0");
        days.type = "number";
        days.min = "0";
        days.oninput = () => {
            plan.access_days = parseInt(days.value || "0") || 0;
            this.renderPlanList();
        };
        accessRow.appendChild(this.editorField("Days", "Extra days added after months. Use this for weekly, trial, or odd-length passes.", days));
        access.appendChild(accessRow);
        const profile = this.input("payments-plan-profile", plan.profile || "Default", "Default");
        profile.oninput = () => {
            plan.profile = profile.value;
        };
        access.appendChild(this.editorField("Profile", "The jfa-go user profile used when a new paid invite is created.", profile));
        this._planEditor.appendChild(access);

        const billing = this.editorSection("Billing", "How Stripe should collect payment for this plan.");
        const recurring = this.switchControl("Recurring subscription", plan.recurring, (checked) => {
            plan.recurring = checked;
            if (checked) {
                plan.stripe_interval = plan.stripe_interval || "month";
                plan.stripe_interval_count = plan.stripe_interval_count || 1;
            }
            this.renderPlans();
        });
        billing.appendChild(this.editorField("Type", "Recurring plans create Stripe subscriptions. One-time plans create a single payment only.", recurring));
        const cadence = document.createElement("div");
        cadence.className = "grid grid-cols-1 sm:grid-cols-[8rem_1fr] gap-2";
        const intervalCount = this.input("payments-plan-interval-count", String(plan.stripe_interval_count || 1), "1");
        intervalCount.type = "number";
        intervalCount.min = "1";
        intervalCount.disabled = !plan.recurring;
        intervalCount.oninput = () => {
            plan.stripe_interval_count = parseInt(intervalCount.value || "1") || 1;
            this.renderPlanList();
        };
        cadence.appendChild(intervalCount);
        cadence.appendChild(this.intervalSelect(plan.stripe_interval || "month", (value) => {
            plan.stripe_interval = value;
            this.renderPlanList();
        }, !plan.recurring));
        billing.appendChild(this.editorField("Charge every", "Stripe billing cadence. Quarterly is 3 + month; yearly is 1 + year.", cadence));
        const planID = this.input("payments-plan-id", plan.id || "", "monthly");
        planID.oninput = () => {
            plan.id = planID.value;
            this.renderPlanList();
        };
        billing.appendChild(this.editorField("Plan ID", "Internal ID written to Stripe metadata. Keep it stable after people start buying this plan.", planID));
        this._planEditor.appendChild(billing);
    };

    private collectPlans = (): PaymentPlan[] => {
        const plans = this._plans.map((plan) => this.cleanPlan(plan));
        if (plans.length == 0) throw new Error("At least one payment plan is required");
        const ids = new Set<string>();
        for (const plan of plans) {
            if (!plan.name) throw new Error("Every plan needs a name");
            if (!plan.id) throw new Error("Every plan needs an ID");
            if (!Number.isFinite(plan.price) || plan.price <= 0) throw new Error("Every plan needs a positive price");
            if ((plan.access_months || 0) + (plan.access_days || 0) <= 0) throw new Error("Every plan needs an access duration");
            if (ids.has(plan.id)) throw new Error(`Duplicate plan ID "${plan.id}"`);
            ids.add(plan.id);
        }
        return plans;
    };

    private cleanPlan = (plan: PaymentPlan): PaymentPlan => {
        const clean: PaymentPlan = {
            id: (plan.id || "").trim(),
            name: (plan.name || "").trim(),
            description: (plan.description || "").trim(),
            enabled: Boolean(plan.enabled),
            price: Math.round(plan.price || 0),
            currency: (plan.currency || "usd").trim().toLowerCase(),
            recurring: Boolean(plan.recurring),
            stripe_interval: plan.recurring ? (plan.stripe_interval || "month") : "",
            stripe_interval_count: plan.recurring ? Math.max(1, Math.round(plan.stripe_interval_count || 1)) : 0,
            access_months: Math.max(0, Math.round(plan.access_months || 0)),
            access_days: Math.max(0, Math.round(plan.access_days || 0)),
            profile: (plan.profile || "Default").trim() || "Default",
        };
        return clean;
    };

    private input = (className: string, value: string, placeholder: string): HTMLInputElement => {
        const input = document.createElement("input");
        input.className = "input ~neutral @low full-width " + className;
        input.value = value;
        input.placeholder = placeholder;
        return input;
    };

    private currencySelect = (value: string, onChange: (value: string) => void): HTMLDivElement => {
        return this.selectControl(["usd", "eur", "gbp", "cad", "aud"].map((currency) => [currency, currency.toUpperCase()]), value || "usd", "payments-plan-currency", onChange);
    };

    private intervalSelect = (value: string, onChange: (value: string) => void, disabled: boolean): HTMLDivElement => {
        return this.selectControl(["day", "week", "month", "year"].map((interval) => [interval, interval]), value || "month", "payments-plan-interval", onChange, disabled);
    };

    private selectControl = (options: string[][], value: string, className: string, onChange: (value: string) => void, disabled: boolean = false): HTMLDivElement => {
        const wrapper = document.createElement("div");
        wrapper.className = "select ~neutral @low full-width";
        const select = document.createElement("select");
        select.className = className;
        select.disabled = disabled;
        for (const optionValue of options) {
            const option = document.createElement("option");
            option.value = optionValue[0];
            option.textContent = optionValue[1];
            select.appendChild(option);
        }
        select.value = value;
        select.onchange = () => onChange(select.value);
        wrapper.appendChild(select);
        return wrapper;
    };

    private editorSection = (heading: string, description: string): HTMLDivElement => {
        const section = document.createElement("div");
        section.className = "flex flex-col gap-3 border-t border-neutral-600/40 pt-4";
        const header = document.createElement("div");
        header.className = "flex flex-col gap-1";
        const title = document.createElement("span");
        title.className = "font-medium";
        title.textContent = heading;
        const desc = document.createElement("span");
        desc.className = "support";
        desc.textContent = description;
        header.append(title, desc);
        section.appendChild(header);
        return section;
    };

    private editorField = (label: string, help: string, control: HTMLElement): HTMLDivElement => {
        const field = document.createElement("div");
        field.className = "label flex flex-col gap-2";
        field.appendChild(this.fieldLabel(label, help));
        field.appendChild(control);
        return field;
    };

    private fieldLabel = (label: string, help: string): HTMLDivElement => {
        const row = document.createElement("div");
        row.className = "flex flex-row gap-2 items-baseline";
        const text = document.createElement("span");
        text.className = "supra";
        text.textContent = label;
        row.appendChild(text);

        const tooltip = document.createElement("tool-tip");
        tooltip.className = "setting-tooltip below-center sm:right";
        tooltip.innerHTML = `
            <i class="icon ri-information-line align-[-0.05rem]"></i>
            <span class="content sm"></span>
        `;
        (tooltip.querySelector(".content") as HTMLElement).textContent = help;
        row.appendChild(tooltip);
        return row;
    };

    private switchControl = (label: string, checked: boolean, onChange: (checked: boolean) => void): HTMLLabelElement => {
        const wrapper = document.createElement("label");
        wrapper.className = "switch";
        const input = document.createElement("input");
        input.type = "checkbox";
        input.checked = checked;
        input.onchange = () => onChange(input.checked);
        const text = document.createElement("span");
        text.textContent = label;
        wrapper.append(input, text);
        return wrapper;
    };

    private planStat = (label: string, value: string, icon: string): HTMLDivElement => {
        const stat = document.createElement("div");
        stat.className = "flex flex-row gap-2 items-center p-2 rounded-md border border-neutral-600/40";
        const iconEl = document.createElement("i");
        iconEl.className = icon;
        const copy = document.createElement("div");
        copy.className = "flex flex-col min-w-0";
        const title = document.createElement("span");
        title.className = "support";
        title.textContent = label;
        const val = document.createElement("span");
        val.className = "font-medium truncate";
        val.textContent = value;
        copy.append(title, val);
        stat.append(iconEl, copy);
        return stat;
    };

    private deleteSelectedPlan = () => {
        if (this._plans.length == 0) return;
        this._plans.splice(this._selectedPlanIndex, 1);
        this._selectedPlanIndex = Math.min(this._selectedPlanIndex, this._plans.length - 1);
        if (this._selectedPlanIndex < 0) this._selectedPlanIndex = 0;
        this.renderPlans();
    };

    private uniquePlanID = (base: string): string => {
        let id = base;
        let i = 2;
        while (this._plans.some((plan) => plan.id == id)) {
            id = `${base}_${i}`;
            i++;
        }
        return id;
    };

    private priceInputValue = (price: number): string => {
        return ((price || 0) / 100).toFixed(2);
    };

    private formatPlanBilling = (plan: PaymentPlan): string => {
        if (!plan.recurring) return "One-time";
        const interval = plan.stripe_interval || "month";
        const count = plan.stripe_interval_count || 1;
        if (count <= 1) return `Every ${interval}`;
        return `Every ${count} ${interval}s`;
    };

    private formatPlanAccess = (plan: PaymentPlan): string => {
        const parts: string[] = [];
        const months = plan.access_months || 0;
        const days = plan.access_days || 0;
        if (months > 0) parts.push(`${months} ${months == 1 ? "month" : "months"}`);
        if (days > 0) parts.push(`${days} ${days == 1 ? "day" : "days"}`);
        return parts.length ? parts.join(" + ") : "No access";
    };

    private renderPayments = (payments: PaymentRecord[]) => {
        this._list.replaceChildren();
        this._empty.classList.toggle("unfocused", payments.length > 0);

        for (const payment of payments) {
            this._list.appendChild(this.paymentRow(payment));
        }
    };

    private paymentRow = (payment: PaymentRecord): HTMLTableRowElement => {
        const tr = document.createElement("tr");
        tr.appendChild(this.textCell(this.formatTime(payment.created)));
        tr.appendChild(this.textCell(payment.target_email || "-"));
        tr.appendChild(this.textCell(payment.plan || "-"));
        tr.appendChild(this.amountCell(payment));

        const statusCell = document.createElement("td");
        statusCell.className = "whitespace-nowrap";
        statusCell.appendChild(this.statusBadge(payment.status));
        if (payment.subscription_status) {
            statusCell.appendChild(document.createTextNode(" "));
            statusCell.appendChild(this.providerStatusBadge("Sub " + this.formatStatus(payment.subscription_status)));
        }
        if (payment.invoice_status) {
            statusCell.appendChild(document.createTextNode(" "));
            statusCell.appendChild(this.providerStatusBadge("Invoice " + this.formatStatus(payment.invoice_status)));
        }
        if (payment.email_status && payment.email_status != "not_applicable") {
            statusCell.appendChild(document.createTextNode(" "));
            statusCell.appendChild(this.emailBadge(payment.email_status));
        }
        if (payment.error) {
            const err = document.createElement("div");
            err.className = "support max-w-[18rem] truncate";
            err.title = payment.error;
            err.textContent = payment.error;
            statusCell.appendChild(err);
        }
        const subscriptionDate = payment.subscription_cancel_at || payment.subscription_canceled_at || payment.subscription_ended_at;
        if (subscriptionDate) {
            const cancel = document.createElement("div");
            cancel.className = "support whitespace-nowrap";
            cancel.textContent =
                (payment.subscription_cancel_at_period_end ? "Cancels " : "Canceled ") + this.formatTime(subscriptionDate);
            statusCell.appendChild(cancel);
        }
        tr.appendChild(statusCell);

        tr.appendChild(this.textCell(payment.invite_code || payment.jellyfin_id || "-"));

        const idCell = this.providerIDCell(payment);
        tr.appendChild(idCell);

        const actionCell = document.createElement("td");
        actionCell.className = "text-right";
        if (this.canCancelSubscription(payment)) {
            const cancel = document.createElement("button");
            cancel.className = "button ~critical @low";
            cancel.title = "Cancel subscription";
            cancel.ariaLabel = "Cancel subscription";
            cancel.innerHTML = `<i class="ri-close-circle-line"></i>`;
            cancel.onclick = () => this.openCancelSubscription(payment);
            actionCell.appendChild(cancel);
        }
        if (payment.invite_code && payment.target_email) {
            const resend = document.createElement("button");
            resend.className = "button ~neutral @low";
            resend.title = "Resend invite";
            resend.ariaLabel = "Resend invite";
            resend.innerHTML = `<i class="ri-mail-send-line"></i>`;
            resend.disabled = payment.email_status == "pending";
            resend.onclick = () => this.resendInvite(payment.id, resend);
            if (actionCell.children.length > 0) actionCell.appendChild(document.createTextNode(" "));
            actionCell.appendChild(resend);
        }
        tr.appendChild(actionCell);
        return tr;
    };

    private canCancelSubscription = (payment: PaymentRecord): boolean => {
        if (!payment.subscription_id) return false;
        const status = payment.subscription_status || payment.status;
        return [
            "canceled",
            "subscription_canceled",
            "subscription_lapsed",
            "refunded",
        ].indexOf(status) == -1;
    };

    private openCancelSubscription = (payment: PaymentRecord) => {
        this._cancelPayment = payment;
        this._cancelSummary.textContent = `${payment.target_email || "Unknown user"} · ${this.shortProviderID(payment.subscription_id)}`;
        (this._cancelForm.querySelector("input[name='payment-subscription-cancel-when'][value='period_end']") as HTMLInputElement).checked = true;
        this._cancelCustom.value = this.datetimeLocalValue(new Date(Date.now() + 24 * 60 * 60 * 1000));
        this._cancelRefund.checked = false;
        window.modals.paymentSubscriptionCancel.show();
    };

    private cancelSubscription = (event: Event) => {
        event.preventDefault();
        if (!this._cancelPayment) return;

        const selected = this._cancelForm.querySelector("input[name='payment-subscription-cancel-when']:checked") as HTMLInputElement;
        const payload = {
            when: selected?.value || "period_end",
            cancel_at: 0,
            refund: this._cancelRefund.checked,
        };
        if (payload.when == "custom") {
            const customDate = new Date(this._cancelCustom.value);
            if (!this._cancelCustom.value || Number.isNaN(customDate.getTime())) {
                window.notifications.customError("paymentSubscriptionCancel", "Choose a valid cancellation date");
                return;
            }
            payload.cancel_at = Math.floor(customDate.getTime() / 1000);
        }
        if (!window.confirm(this.cancelConfirmationText(payload.when, payload.cancel_at, payload.refund))) {
            return;
        }

        addLoader(this._cancelSubmit);
        _post(`/payments/${encodeURIComponent(this._cancelPayment.id)}/subscription/cancel`, payload, (req: XMLHttpRequest) => {
            if (req.readyState != 4) return;
            removeLoader(this._cancelSubmit);
            if (req.status != 200) {
                window.notifications.customError("paymentSubscriptionCancel", req.response?.error || "Failed to cancel subscription");
                return;
            }

            const result = req.response as CancelSubscriptionResponse;
            window.modals.paymentSubscriptionCancel.close();
            window.notifications.customSuccess(
                "paymentSubscriptionCancel",
                result.refund_id ? "Subscription canceled and refund created" : "Subscription cancellation updated",
            );
            this.loadPayments();
        }, true);
    };

    private cancelConfirmationText = (when: string, cancelAt: number, refund: boolean): string => {
        const target = this._cancelPayment?.target_email || this.shortProviderID(this._cancelPayment?.subscription_id || "");
        const targetText = target ? ` for ${target}` : "";
        let action = "schedule the subscription to cancel at the end of the current billing period";
        if (when == "now") {
            action = "end the subscription immediately";
        } else if (when == "custom") {
            action = `schedule the subscription to cancel on ${this.formatTime(cancelAt)}`;
        }
        const refundText = refund ? " This will also try to refund the latest refundable payment." : "";
        return `Are you sure you want to ${action}${targetText}?${refundText}`;
    };

    private resendInvite = (paymentID: string, button: HTMLButtonElement) => {
        addLoader(button);
        _post("/payments/" + encodeURIComponent(paymentID) + "/resend", null, (req: XMLHttpRequest) => {
            if (req.readyState != 4) return;
            removeLoader(button);
            if (req.status == 200) {
                window.notifications.customSuccess("paymentInviteResent", "Invite queued");
                this.loadPayments();
            } else {
                window.notifications.customError("paymentInviteResent", "Failed to queue invite");
            }
        });
    };

    private textCell = (text: string): HTMLTableCellElement => {
        const td = document.createElement("td");
        td.textContent = text;
        return td;
    };

    private amountCell = (payment: PaymentRecord): HTMLTableCellElement => {
        const td = this.textCell(this.formatMoney(payment.amount, payment.currency));
        if (payment.refunded_amount) {
            const refund = document.createElement("div");
            refund.className = "support";
            refund.textContent = "Refunded " + this.formatMoney(payment.refunded_amount, payment.currency);
            td.appendChild(refund);
        }
        return td;
    };

    private providerIDCell = (payment: PaymentRecord): HTMLTableCellElement => {
        const td = document.createElement("td");
        td.classList.add("max-w-[16rem]");
        const links = this.providerLinks(payment);
        if (links.length == 0) {
            td.textContent = "-";
            return td;
        }
        const wrap = document.createElement("div");
        wrap.className = "flex flex-row flex-wrap gap-1";
        for (const link of links) {
            wrap.appendChild(this.providerLinkBadge(link.label, link.id, link.url));
        }
        td.appendChild(wrap);
        return td;
    };

    private providerLinks = (payment: PaymentRecord): { label: string; id: string; url: string }[] => {
        if ((payment.provider || "").toLowerCase() != "stripe") {
            const id = payment.provider_payment_id || payment.id;
            return id ? [{ label: payment.provider || "Provider", id: id, url: "" }] : [];
        }

        const links: { label: string; id: string; url: string }[] = [];
        const add = (label: string, id: string | undefined, path: string) => {
            if (!id || links.some((link) => link.id == id)) return;
            links.push({ label: label, id: id, url: this.stripeDashboardURL(payment, path) });
        };

        if (payment.subscription_id) {
            add("Subscription", payment.subscription_id, `subscriptions/${payment.subscription_id}`);
            return links;
        }
        add("Customer", payment.customer_id, `customers/${payment.customer_id}`);
        add("Payment", payment.payment_intent_id || payment.charge_id, `payments/${payment.payment_intent_id || payment.charge_id}`);
        add("Invoice", payment.invoice_id, `invoices/${payment.invoice_id}`);
        if ((payment.provider_payment_id || payment.id || "").startsWith("cs_")) {
            const sessionID = payment.provider_payment_id || payment.id;
            add("Checkout", sessionID, `checkout/sessions/${sessionID}`);
        }
        if (links.length == 0 && payment.provider_payment_id) {
            add("Stripe", payment.provider_payment_id, `search?query=${encodeURIComponent(payment.provider_payment_id)}`);
        }
        return links;
    };

    private providerLinkBadge = (label: string, id: string, url: string): HTMLElement => {
        const el = document.createElement(url ? "a" : "span");
        el.className = "button ~neutral @low dark:~d_neutral font-mono text-sm px-2 py-1";
        el.textContent = `${label} ${this.shortProviderID(id)}`;
        el.title = id;
        if (url) {
            const link = el as HTMLAnchorElement;
            link.href = url;
            link.target = "_blank";
            link.rel = "noopener noreferrer";
            link.ariaLabel = `Open ${label} ${id} in Stripe`;
        }
        return el;
    };

    private stripeDashboardURL = (payment: PaymentRecord, path: string): string => {
        const mode = payment.provider_live_mode ? "" : "/test";
        return `https://dashboard.stripe.com${mode}/${path}`;
    };

    private shortProviderID = (id: string): string => {
        if (!id || id.length <= 16) return id || "";
        const parts = id.split("_");
        if (parts.length >= 3 && (parts[1] == "test" || parts[1] == "live")) {
            return `${parts[0]}_${parts[1]}…${id.slice(-4)}`;
        }
        if (parts.length >= 2) {
            return `${parts[0]}_${parts[1].slice(0, 4)}…${id.slice(-4)}`;
        }
        return `${id.slice(0, 8)}…${id.slice(-4)}`;
    };

    private datetimeLocalValue = (date: Date): string => {
        const pad = (value: number): string => value < 10 ? "0" + value : String(value);
        return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(date.getHours())}:${pad(date.getMinutes())}`;
    };

    private statusBadge = (status: string): HTMLSpanElement => {
        const tones = {
            checkout_created: "~neutral",
            checkout_expired: "~warning",
            paid: "~info",
            fulfilled: "~positive",
            email_sent: "~positive",
            email_failed: "~critical",
            payment_canceled: "~warning",
            refunded: "~warning",
            partially_refunded: "~warning",
            subscription_canceling: "~warning",
            subscription_canceled: "~warning",
            subscription_past_due: "~critical",
            subscription_lapsed: "~critical",
            failed: "~critical",
            needs_review: "~warning",
        };
        return this.badge(this.formatStatus(status), tones[status] || "~neutral");
    };

    private providerStatusBadge = (label: string): HTMLSpanElement => {
        return this.badge(label, "~neutral");
    };

    private emailBadge = (status: string): HTMLSpanElement => {
        const tones = {
            pending: "~info",
            sent: "~positive",
            failed: "~critical",
            disabled: "~warning",
            not_started: "~neutral",
        };
        return this.badge("Email " + this.formatStatus(status), tones[status] || "~neutral");
    };

    private badge = (text: string, tone: string): HTMLSpanElement => {
        const span = document.createElement("span");
        span.className = "badge " + tone;
        span.textContent = text;
        return span;
    };

    private formatStatus = (status: string): string => {
        if (!status) return "-";
        return status
            .split("_")
            .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
            .join(" ");
    };

    private formatTime = (unixSeconds: number): string => {
        if (!unixSeconds) return "-";
        return toDateString(new Date(unixSeconds * 1000));
    };

    private formatMoney = (amount: number, currency: string): string => {
        if (!amount || !currency) return "-";
        try {
            return new Intl.NumberFormat(window.language || undefined, {
                style: "currency",
                currency: currency.toUpperCase(),
            }).format(amount / 100);
        } catch (_) {
            return `${(amount / 100).toFixed(2)} ${currency.toUpperCase()}`;
        }
    };
}
