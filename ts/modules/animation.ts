import { rmAttr, addAttr } from "../modules/common.js";

interface aWindow extends Window {
    rotateButton(el: HTMLElement): void;
}

declare var window: aWindow;

// Used for animation on theme change
const whichTransitionEvent = (): string => {
    const el = document.createElement('fakeElement');
    const transitions = {
        'transition': 'transitionend',
        'OTransition': 'oTransitionEnd',
        'MozTransition': 'transitionend',
        'WebkitTransition': 'webkitTransitionEnd'
    };
    for (const t in transitions) {
        if (el.style[t] !== undefined) {
            return transitions[t];
        }
    }
    return '';
};

var transitionEndEvent = whichTransitionEvent();

// Toggles between light and dark themes
const _toggleCSS = (): void => {
    const els: NodeListOf<HTMLLinkElement> = document.querySelectorAll('link[rel="stylesheet"][type="text/css"]');
    let cssEl = 0;
    let remove = false;
    if (els.length != 1) {
        cssEl = 1;
        remove = true
    }
    let href: string = "bs" + window.bsVersion;
    if (!els[cssEl].href.includes(href + "-jf")) {
        href += "-jf";
    }
    href += ".css";
    let newEl = els[cssEl].cloneNode(true) as HTMLLinkElement;
    newEl.href = href;
    els[cssEl].parentNode.insertBefore(newEl, els[cssEl].nextSibling);
    if (remove) {
        els[0].remove();
    }
    document.cookie = "css=" + href;
}

// Toggles between light and dark themes, but runs animation if window small enough.
window.buttonWidth = 0;
export const toggleCSS = (el: HTMLElement): void => {
    const switchToColor = window.getComputedStyle(document.body, null).backgroundColor;
    // Max page width for animation to take place
    let maxWidth = 1500;
    if (window.innerWidth < maxWidth) {
        // Calculate minimum radius to cover screen
        const radius = Math.sqrt(Math.pow(window.innerWidth, 2) + Math.pow(window.innerHeight, 2));
        const currentRadius = el.getBoundingClientRect().width / 2;
        const scale = radius / currentRadius;
        window.buttonWidth = +window.getComputedStyle(el, null).width;
        document.body.classList.remove('smooth-transition');
        el.style.transform = `scale(${scale})`;
        el.style.color = switchToColor;
        el.addEventListener(transitionEndEvent, function (): void {
            if (this.style.transform.length != 0) {
                _toggleCSS();
                this.style.removeProperty('transform');
                document.body.classList.add('smooth-transition');
            }
        }, false);
    } else {
        _toggleCSS();
        el.style.color = switchToColor;
    }
};

window.rotateButton = (el: HTMLElement): void => {
    if (el.classList.contains("rotated")) {
        rmAttr(el, "rotated")
        addAttr(el, "not-rotated");
    } else {
        rmAttr(el, "not-rotated");
        addAttr(el, "rotated");
    }
};
