"use strict";

const DEFAULT_MAX_SIZE = 5 * 1024 * 1024;

const state = {
    files: [],
    maxSize: DEFAULT_MAX_SIZE,
    pendingReads: 0,
    objectUrls: new Set(),
    toastTimer: null,
    shares: new Map(),
    receiptSource: null
};

let wrapEditor = null;
let unwrapEditor = null;

const elements = {
    body: document.body,
    themeToggle: document.getElementById("themeToggle"),
    dropZone: document.getElementById("dropZone"),
    attachButton: document.getElementById("attachButton"),
    fileInput: document.getElementById("fileInput"),
    attachmentList: document.getElementById("attachmentList"),
    sizeLabel: document.getElementById("sizeLabel"),
    sizeTrack: document.getElementById("sizeTrack"),
    sizeBar: document.getElementById("sizeBar"),
    ttl: document.getElementById("ttl"),
    ttlPreview: document.getElementById("ttlPreview"),
    ttlPresets: document.getElementById("ttlPresets"),
    wrapButton: document.getElementById("wrapButton"),
    wrapError: document.getElementById("wrapError"),
    wrapResult: document.getElementById("wrapResult"),
    shareList: document.getElementById("shareList"),
    liveReceiptState: document.getElementById("liveReceiptState"),
    liveReceiptLabel: document.getElementById("liveReceiptLabel"),
    unwrapInput: document.getElementById("unwrapInput"),
    toggleTokenVisibility: document.getElementById("toggleTokenVisibility"),
    pasteButton: document.getElementById("pasteButton"),
    sharedTokenNotice: document.getElementById("sharedTokenNotice"),
    unwrapButton: document.getElementById("unwrapButton"),
    unwrapError: document.getElementById("unwrapError"),
    unwrapPanel: document.getElementById("unwrapPanel"),
    unwrapResult: document.getElementById("unwrapResult"),
    openedResultTitle: document.getElementById("openedResultTitle"),
    unwrappedTextBlock: document.getElementById("unwrappedTextBlock"),
    downloadList: document.getElementById("downloadList"),
    unwrapMetaDetails: document.getElementById("unwrapMetaDetails"),
    unwrapDetails: document.getElementById("unwrapDetails"),
    copyUnwrappedText: document.getElementById("copyUnwrappedText"),
    openAnotherButton: document.getElementById("openAnotherButton"),
    toast: document.getElementById("toast"),
    vaultStatus: document.getElementById("vaultStatus"),
    vaultStatusDot: document.getElementById("vaultStatusDot"),
    vaultStatusLabel: document.getElementById("vaultStatusLabel"),
    vaultStatusPanel: document.getElementById("vaultStatusPanel"),
    appVersion: document.getElementById("appVersion"),
    githubLink: document.getElementById("githubLink")
};

function preferredTheme() {
    try {
        const saved = localStorage.getItem("vault-wrapper-theme");
        if (saved === "dark" || saved === "light") return saved;
    } catch (_) {
        // Storage can be unavailable in privacy modes; use the system setting.
    }
    return window.matchMedia?.("(prefers-color-scheme: dark)").matches ? "dark" : "light";
}

function applyTheme(theme) {
    const isDark = theme === "dark";
    elements.body.classList.toggle("dark-mode", isDark);
    elements.themeToggle.setAttribute("aria-label", isDark ? "Use light theme" : "Use dark theme");
    elements.themeToggle.title = isDark ? "Use light theme" : "Use dark theme";
    wrapEditor?.setOption("theme", isDark ? "dracula" : "default");
    unwrapEditor?.setOption("theme", isDark ? "dracula" : "default");
}

applyTheme(preferredTheme());

wrapEditor = CodeMirror(document.getElementById("wrapInput"), {
    lineNumbers: true,
    lineWrapping: true,
    mode: "javascript",
    theme: elements.body.classList.contains("dark-mode") ? "dracula" : "default",
    dragDrop: false,
    placeholder: "Paste a password, configuration, note, or code snippet…"
});

unwrapEditor = CodeMirror(document.getElementById("unwrapEditor"), {
    lineNumbers: false,
    lineWrapping: true,
    mode: null,
    theme: elements.body.classList.contains("dark-mode") ? "dracula" : "default",
    readOnly: true
});

wrapEditor.getInputField().setAttribute("aria-label", "Secret text or code");
wrapEditor.getInputField().setAttribute("autocomplete", "off");
wrapEditor.getInputField().setAttribute("spellcheck", "false");
unwrapEditor.getInputField().setAttribute("aria-label", "Unwrapped text content");

function formatBytes(bytes) {
    if (!Number.isFinite(bytes) || bytes <= 0) return "0 B";
    const units = ["B", "KB", "MB", "GB"];
    const index = Math.min(Math.floor(Math.log(bytes) / Math.log(1024)), units.length - 1);
    const value = bytes / Math.pow(1024, index);
    const digits = index === 0 ? 0 : value >= 10 ? 1 : 2;
    return `${value.toFixed(digits)} ${units[index]}`;
}

function textSize() {
    return new TextEncoder().encode(wrapEditor.getValue()).length;
}

function filesSize() {
    return state.files.reduce((total, file) => total + file.size, 0);
}

function payloadSize() {
    return textSize() + filesSize();
}

function updateSizeMeter() {
    const size = payloadSize();
    const percentage = Math.min(100, (size / state.maxSize) * 100 || 0);
    elements.sizeLabel.textContent = `${formatBytes(size)} of ${formatBytes(state.maxSize)}`;
    elements.sizeBar.style.width = `${percentage}%`;
    elements.sizeTrack.setAttribute("aria-valuenow", String(Math.round(percentage)));
    elements.sizeTrack.classList.toggle("warning", percentage >= 75 && percentage < 100);
    elements.sizeTrack.classList.toggle("error", size > state.maxSize);
}

function fileIcon() {
    const icon = document.createElement("span");
    icon.className = "attachment-icon";
    icon.textContent = "↥";
    icon.setAttribute("aria-hidden", "true");
    return icon;
}

function renderAttachments() {
    elements.attachmentList.replaceChildren();
    state.files.forEach((file, index) => {
        const row = document.createElement("div");
        row.className = "attachment-chip";
        row.appendChild(fileIcon());

        const name = document.createElement("span");
        name.className = "attachment-name";
        name.textContent = file.name;
        name.title = file.name;
        row.appendChild(name);

        const size = document.createElement("span");
        size.className = "attachment-size";
        size.textContent = formatBytes(file.size);
        row.appendChild(size);

        const remove = document.createElement("button");
        remove.type = "button";
        remove.className = "remove-file";
        remove.textContent = "×";
        remove.setAttribute("aria-label", `Remove ${file.name}`);
        remove.addEventListener("click", () => {
            state.files.splice(index, 1);
            renderAttachments();
            updateSizeMeter();
        });
        row.appendChild(remove);
        elements.attachmentList.appendChild(row);
    });
}

function arrayBufferToBase64(buffer) {
    const bytes = new Uint8Array(buffer);
    const chunks = [];
    const chunkSize = 0x8000;
    for (let offset = 0; offset < bytes.length; offset += chunkSize) {
        chunks.push(String.fromCharCode(...bytes.subarray(offset, offset + chunkSize)));
    }
    return btoa(chunks.join(""));
}

let fileQueue = Promise.resolve();

function queueFiles(fileList) {
    const files = Array.from(fileList || []);
    if (!files.length) return;
    state.pendingReads += files.length;
    fileQueue = fileQueue.then(async () => {
        const skipped = [];
        for (const file of files) {
            const remaining = state.maxSize - payloadSize();
            if (file.size > remaining) {
                skipped.push(file.name);
                state.pendingReads -= 1;
                continue;
            }
            try {
                const data = arrayBufferToBase64(await file.arrayBuffer());
                state.files.push({
                    isFile: true,
                    name: file.name,
                    type: file.type || "application/octet-stream",
                    data,
                    size: file.size
                });
            } catch (_) {
                skipped.push(file.name);
            } finally {
                state.pendingReads -= 1;
            }
            renderAttachments();
            updateSizeMeter();
        }
        if (skipped.length) {
            showAlert(elements.wrapError, `Could not attach ${skipped.join(", ")}. The combined payload must stay under ${formatBytes(state.maxSize)}.`);
        }
    });
}

function showAlert(element, message) {
    element.textContent = message;
    element.hidden = false;
}

function clearAlert(element) {
    element.textContent = "";
    element.hidden = true;
}

function setBusy(button, busy, busyLabel) {
    if (!button.dataset.defaultLabel) {
        button.dataset.defaultLabel = button.querySelector("span")?.textContent || button.textContent;
    }
    button.disabled = busy;
    button.classList.toggle("loading", busy);
    button.setAttribute("aria-busy", String(busy));
    const label = button.querySelector("span");
    if (label) label.textContent = busy ? busyLabel : button.dataset.defaultLabel;
}

async function responseError(response, fallback) {
    const message = (await response.text()).trim();
    if (response.status === 413) return `The payload exceeds the ${formatBytes(state.maxSize)} limit.`;
    if (response.status === 429) return "Too many requests. Wait a moment and try again.";
    return message || fallback;
}

function setLiveReceiptState(label, status) {
    elements.liveReceiptLabel.textContent = label;
    elements.liveReceiptState.dataset.state = status;
}

function shareStatusLabel(status) {
    if (status === "retrieved") return "Retrieved";
    if (status === "expired") return "Expired";
    if (status === "untracked") return "Status unavailable";
    return "Waiting";
}

function countdownText(milliseconds) {
    const seconds = Math.max(0, Math.ceil(milliseconds / 1000));
    const days = Math.floor(seconds / 86400);
    const hours = Math.floor((seconds % 86400) / 3600);
    const minutes = Math.floor((seconds % 3600) / 60);
    const remainder = seconds % 60;
    if (days > 0) return `${days}d ${String(hours).padStart(2, "0")}h`;
    if (hours > 0) return `${hours}:${String(minutes).padStart(2, "0")}:${String(remainder).padStart(2, "0")}`;
    return `${String(minutes).padStart(2, "0")}:${String(remainder).padStart(2, "0")}`;
}

function durationLabel(seconds) {
    if (!Number.isFinite(seconds) || seconds <= 0) return "seconds";
    const parts = [];
    const days = Math.floor(seconds / 86400);
    const hours = Math.floor((seconds % 86400) / 3600);
    const minutes = Math.floor((seconds % 3600) / 60);
    const remainder = seconds % 60;
    if (days) parts.push(`${days}d`);
    if (hours) parts.push(`${hours}h`);
    if (minutes) parts.push(`${minutes}m`);
    if (remainder || parts.length === 0) parts.push(`${remainder}s`);
    return `seconds · ${parts.slice(0, 2).join(" ")}`;
}

function updateTtlPreview() {
    elements.ttlPreview.textContent = durationLabel(Number.parseInt(elements.ttl.value, 10));
}

function maybeCloseReceiptStream() {
    const hasWaiting = Array.from(state.shares.values()).some((share) => share.status === "waiting");
    if (hasWaiting || !state.receiptSource) return;
    state.receiptSource.close();
    state.receiptSource = null;
    setLiveReceiptState("Updates complete", "idle");
}

function updateShareStatus(receiptID, status, expiresAt) {
    const share = state.shares.get(receiptID);
    if (!share) return;
    const previousStatus = share.status;
    if (expiresAt) share.expiresAt = new Date(expiresAt);
    share.status = status;
    share.card.dataset.status = status;
    share.statusNode.textContent = shareStatusLabel(status);
    if (status === "retrieved") share.headingNode.textContent = "Secret retrieved";
    else if (status === "expired") share.headingNode.textContent = "Link expired";
    else share.headingNode.textContent = "Ready to share";
    const settled = status === "retrieved" || status === "expired";
    share.copyButtons.forEach((button) => { button.disabled = settled; });
    share.card.querySelectorAll("input").forEach((input) => { input.disabled = settled; });
    renderShareTimer(share);
    if (status !== previousStatus) {
        share.card.classList.remove("status-change");
        void share.card.offsetWidth;
        share.card.classList.add("status-change");
        window.setTimeout(() => share.card.classList.remove("status-change"), 700);
    }
    if (status === "retrieved" && previousStatus !== "retrieved") showToast("A shared secret was retrieved");
    maybeCloseReceiptStream();
}

function ensureReceiptStream() {
    if (state.receiptSource) return;
    if (!window.EventSource) {
        setLiveReceiptState("Updates unavailable", "error");
        return;
    }
    setLiveReceiptState("Connecting", "connecting");
    const source = new EventSource("/api/events");
    state.receiptSource = source;
    source.addEventListener("open", () => setLiveReceiptState("Live updates", "connected"));
    source.addEventListener("receipt", (event) => {
        try {
            const update = JSON.parse(event.data);
            if (typeof update.id === "string" && typeof update.status === "string") {
                updateShareStatus(update.id, update.status, update.expires_at);
            }
        } catch (_) {
            // Ignore malformed stream events and wait for the next snapshot.
        }
    });
    source.addEventListener("error", () => {
        if (state.receiptSource === source) setLiveReceiptState("Reconnecting", "connecting");
    });
}

function makeCopyField(label, value, actionLabel) {
    const field = document.createElement("div");
    field.className = "copy-field";
    const title = document.createElement("span");
    title.className = "copy-field-title";
    title.textContent = label;
    const control = document.createElement("span");
    control.className = "copy-control";
    const input = document.createElement("input");
    input.type = "text";
    input.readOnly = true;
    input.spellcheck = false;
    input.setAttribute("aria-label", label);
    input.value = value;
    const button = document.createElement("button");
    button.type = "button";
    button.textContent = actionLabel;
    let resetTimer = null;
    button.addEventListener("click", async () => {
        try {
            await copyText(value, `${label} copied`);
            window.clearTimeout(resetTimer);
            button.textContent = "Copied";
            control.classList.add("is-copied");
            resetTimer = window.setTimeout(() => {
                button.textContent = actionLabel;
                control.classList.remove("is-copied");
            }, 1400);
        } catch (_) {
            showToast("Clipboard is unavailable");
        }
    });
    control.append(input, button);
    field.append(title, control);
    return { field, button };
}

function addShare(result, shareUrl, ttl) {
    const receipt = result.receipt && typeof result.receipt === "object" ? result.receipt : null;
    const receiptID = typeof receipt?.id === "string" ? receipt.id : `untracked-${Date.now()}`;
    const expiresAt = receipt?.expires_at ? new Date(receipt.expires_at) : new Date(Date.now() + ttl * 1000);
    const status = receipt ? String(receipt.status || "waiting") : "untracked";

    const card = document.createElement("article");
    card.className = "share-card";
    card.dataset.status = status;

    const header = document.createElement("div");
    header.className = "share-card-header";
    const identity = document.createElement("div");
    identity.className = "share-card-identity";
    const statusMark = document.createElement("span");
    statusMark.className = "share-status-mark";
    statusMark.setAttribute("aria-hidden", "true");
    const title = document.createElement("div");
    const kicker = document.createElement("span");
    kicker.className = "share-kicker";
    kicker.textContent = "One-time link";
    const heading = document.createElement("h3");
    heading.textContent = "Ready to share";
    title.append(heading, kicker);
    identity.append(statusMark, title);
    const statusNode = document.createElement("span");
    statusNode.className = "share-status";
    statusNode.textContent = shareStatusLabel(status);
    const time = document.createElement("div");
    time.className = "share-time";
    const countdownNode = document.createElement("strong");
    countdownNode.className = "share-countdown";
    const timeLabelNode = document.createElement("span");
    timeLabelNode.className = "share-time-label";
    time.append(countdownNode, timeLabelNode);
    header.append(identity, statusNode, time);

    const linkField = makeCopyField("Shareable link", shareUrl, "Copy");
    linkField.field.classList.add("share-link-field");

    const details = document.createElement("details");
    details.className = "share-details";
    const summary = document.createElement("summary");
    summary.textContent = "Token and receipt";
    const tokenField = makeCopyField("Raw token", result.token, "Copy");
    const receiptData = document.createElement("pre");
    receiptData.textContent = JSON.stringify(result.details || {}, null, 2);
    details.append(summary, tokenField.field, receiptData);

    const progress = document.createElement("span");
    progress.className = "share-progress";
    progress.setAttribute("aria-hidden", "true");
    const progressFill = document.createElement("span");
    progress.appendChild(progressFill);

    card.append(header, linkField.field, details, progress);
    elements.shareList.prepend(card);
    elements.wrapResult.hidden = false;

    state.shares.set(receiptID, {
        card,
        status,
        headingNode: heading,
        statusNode,
        countdownNode,
        timeLabelNode,
        progressNode: progressFill,
        expiresAt,
        totalMs: Math.max(1000, expiresAt.getTime() - Date.now()),
        copyButtons: [linkField.button, tokenField.button]
    });
    updateShareCountdowns();
    if (status === "waiting") ensureReceiptStream();
    else if (status === "untracked") setLiveReceiptState("Updates unavailable", "error");
    else maybeCloseReceiptStream();
    card.scrollIntoView({ behavior: "smooth", block: "nearest" });
}

function renderShareTimer(share) {
    if (share.status === "retrieved") {
        share.countdownNode.textContent = "Done";
        share.timeLabelNode.textContent = "token consumed";
        share.progressNode.style.transform = "scaleX(0)";
        share.card.dataset.urgency = "settled";
        return;
    }
    if (share.status === "expired") {
        share.countdownNode.textContent = "00:00";
        share.timeLabelNode.textContent = "expired";
        share.progressNode.style.transform = "scaleX(0)";
        share.card.dataset.urgency = "settled";
        return;
    }

    const remainingMs = Math.max(0, share.expiresAt.getTime() - Date.now());
    const ratio = Math.max(0, Math.min(1, remainingMs / share.totalMs));
    share.countdownNode.textContent = countdownText(remainingMs);
    share.timeLabelNode.textContent = "remaining";
    share.progressNode.style.transform = `scaleX(${ratio})`;
    share.card.dataset.urgency = remainingMs <= 60000 ? "urgent" : remainingMs <= 300000 ? "soon" : "normal";
    share.timeLabelNode.parentElement.title = `Expires ${share.expiresAt.toLocaleString()}`;
}

function updateShareCountdowns() {
    state.shares.forEach((share, receiptID) => {
        if (share.status === "waiting" && share.expiresAt.getTime() <= Date.now()) {
            updateShareStatus(receiptID, "expired");
            return;
        }
        renderShareTimer(share);
    });
}

async function createSecret() {
    clearAlert(elements.wrapError);

    if (state.pendingReads > 0) {
        showAlert(elements.wrapError, "Files are still being prepared. Try again in a moment.");
        return;
    }

    const text = wrapEditor.getValue();
    const size = payloadSize();
    if (!text.trim() && state.files.length === 0) {
        showAlert(elements.wrapError, "Add some text or at least one file before creating a link.");
        return;
    }
    if (size > state.maxSize) {
        showAlert(elements.wrapError, `The combined payload exceeds the ${formatBytes(state.maxSize)} limit.`);
        return;
    }

    const ttl = Number.parseInt(elements.ttl.value, 10);
    if (!Number.isSafeInteger(ttl) || ttl <= 0) {
        showAlert(elements.wrapError, "Expiry must be a positive number of seconds.");
        elements.ttl.focus();
        return;
    }

    const data = {};
    if (text.trim()) data.text = text;
    if (state.files.length) data.files = state.files;

    setBusy(elements.wrapButton, true, "Sealing secret…");
    try {
        const response = await fetch("/wrap", {
            method: "POST",
            credentials: "same-origin",
            headers: { "Content-Type": "application/json", "X-Receipt-Tracking": "sse" },
            body: JSON.stringify({ data, ttl: String(ttl) })
        });
        if (!response.ok) throw new Error(await responseError(response, "Vault could not wrap the secret."));

        const result = await response.json();
        if (!result.token) throw new Error("Vault returned an incomplete wrapping response.");

        const shareUrl = new URL(window.location.origin + window.location.pathname);
        shareUrl.hash = new URLSearchParams({ token: result.token }).toString();
        addShare(result, shareUrl.toString(), ttl);

        wrapEditor.setValue("");
        state.files = [];
        renderAttachments();
        updateSizeMeter();
        showToast("Secure link created");
    } catch (error) {
        showAlert(elements.wrapError, error.message || "Could not create the secure link.");
    } finally {
        setBusy(elements.wrapButton, false, "");
    }
}

function revokeObjectUrls() {
    state.objectUrls.forEach((url) => URL.revokeObjectURL(url));
    state.objectUrls.clear();
}

function base64ToBlob(base64, type) {
    const binary = atob(base64);
    const bytes = new Uint8Array(binary.length);
    for (let index = 0; index < binary.length; index += 1) bytes[index] = binary.charCodeAt(index);
    return new Blob([bytes], { type: type || "application/octet-stream" });
}

function renderDownloads(files) {
    revokeObjectUrls();
    elements.downloadList.replaceChildren();
    if (!Array.isArray(files)) return 0;

    let rendered = 0;
    files.forEach((file) => {
        if (!file || typeof file.data !== "string") return;
        try {
            const url = URL.createObjectURL(base64ToBlob(file.data, file.type));
            state.objectUrls.add(url);
            const link = document.createElement("a");
            link.className = "download-item";
            link.href = url;
            link.download = typeof file.name === "string" ? file.name : "download";

            const icon = fileIcon();
            icon.className = "download-icon";
            icon.textContent = "↓";
            link.appendChild(icon);

            const name = document.createElement("span");
            name.className = "download-name";
            name.textContent = link.download;
            link.appendChild(name);

            const size = document.createElement("span");
            size.className = "download-size";
            size.textContent = formatBytes(Number(file.size) || 0);
            link.appendChild(size);

            const action = document.createElement("span");
            action.className = "download-action";
            action.textContent = "Download";
            link.appendChild(action);
            elements.downloadList.appendChild(link);
            rendered += 1;
        } catch (_) {
            // A malformed attachment should not hide any valid text or files.
        }
    });
    return rendered;
}

function clearSharedTokenFromAddress() {
    const url = new URL(window.location.href);
    url.searchParams.delete("token");
    const hash = new URLSearchParams(url.hash.slice(1));
    hash.delete("token");
    url.hash = hash.toString();
    const next = `${url.pathname}${url.search}${url.hash}`;
    window.history.replaceState(null, "", next);
}

function resetOpenedSecret(focusInput = true) {
    elements.body.classList.remove("showing-opened-secret");
    elements.unwrapPanel.classList.remove("has-opened-secret");
    elements.unwrapResult.hidden = true;
    elements.unwrappedTextBlock.hidden = true;
    elements.unwrapMetaDetails.hidden = true;
    elements.unwrapMetaDetails.open = false;
    elements.unwrapDetails.textContent = "";
    elements.unwrapInput.value = "";
    elements.unwrapInput.type = "password";
    elements.toggleTokenVisibility.setAttribute("aria-label", "Show token");
    elements.toggleTokenVisibility.title = "Show token";
    elements.sharedTokenNotice.hidden = true;
    unwrapEditor.setValue("");
    renderDownloads([]);
    clearAlert(elements.unwrapError);
    if (focusInput) setTimeout(() => elements.unwrapInput.focus(), 0);
}

async function openSecret() {
    clearAlert(elements.unwrapError);
    elements.unwrapResult.hidden = true;
    const token = elements.unwrapInput.value.trim();
    if (!token) {
        showAlert(elements.unwrapError, "Paste a wrapped token before opening the secret.");
        elements.unwrapInput.focus();
        return;
    }

    setBusy(elements.unwrapButton, true, "Opening secret…");
    try {
        const response = await fetch("/unwrap", {
            method: "POST",
            credentials: "same-origin",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ token })
        });
        if (!response.ok) {
            if (response.status === 404 || response.status === 410) {
                throw new Error("This token is invalid, expired, or has already been used.");
            }
            throw new Error(await responseError(response, "Vault could not open the secret."));
        }

        const result = await response.json();
        const payload = result.data && typeof result.data === "object" ? result.data : {};
        const hasText = typeof payload.text === "string";
        if (hasText) {
            unwrapEditor.setValue(payload.text);
            elements.unwrappedTextBlock.hidden = false;
            setTimeout(() => unwrapEditor.refresh(), 0);
        } else {
            unwrapEditor.setValue("");
            elements.unwrappedTextBlock.hidden = true;
        }
        const fileCount = renderDownloads(payload.files);
        if (!hasText && fileCount === 0) throw new Error("The retrieved secret did not contain readable data.");

        if (result.wrapping_info) {
            elements.unwrapDetails.textContent = JSON.stringify(result.wrapping_info, null, 2);
            elements.unwrapMetaDetails.hidden = false;
        } else {
            elements.unwrapDetails.textContent = "";
            elements.unwrapMetaDetails.hidden = true;
        }

        elements.unwrapInput.value = "";
        elements.sharedTokenNotice.hidden = true;
        elements.body.classList.add("showing-opened-secret");
        elements.unwrapPanel.classList.add("has-opened-secret");
        elements.unwrapResult.hidden = false;
        clearSharedTokenFromAddress();
        elements.unwrapResult.scrollIntoView({ behavior: "smooth", block: "nearest" });
        elements.openedResultTitle.focus({ preventScroll: true });
        showToast("Secret opened — token consumed");
    } catch (error) {
        showAlert(elements.unwrapError, error.message || "Could not open the secret.");
    } finally {
        setBusy(elements.unwrapButton, false, "");
    }
}

function selectTab(panelId, focus = false) {
    const buttons = Array.from(document.querySelectorAll("[role=tab]"));
    buttons.forEach((button) => {
        const selected = button.dataset.tab === panelId;
        button.classList.toggle("active", selected);
        button.setAttribute("aria-selected", String(selected));
        button.tabIndex = selected ? 0 : -1;
        const panel = document.getElementById(button.dataset.tab);
        panel.hidden = !selected;
        panel.classList.toggle("active", selected);
        if (selected && focus) button.focus();
    });
    elements.body.classList.toggle(
        "showing-opened-secret",
        panelId === "unwrapPanel" && elements.unwrapPanel.classList.contains("has-opened-secret")
    );
    if (panelId === "wrapPanel") setTimeout(() => wrapEditor.refresh(), 0);
    if (panelId === "unwrapPanel") setTimeout(() => unwrapEditor.refresh(), 0);
}

function sharedTokenFromUrl() {
    const url = new URL(window.location.href);
    const fragmentToken = new URLSearchParams(url.hash.slice(1)).get("token");
    const legacyToken = url.searchParams.get("token");
    const token = fragmentToken || legacyToken;
    if (legacyToken) {
        url.searchParams.delete("token");
        url.hash = new URLSearchParams({ token: legacyToken }).toString();
        window.history.replaceState(null, "", `${url.pathname}${url.search}${url.hash}`);
    }
    return token;
}

async function copyText(text, message = "Copied to clipboard") {
    if (!text) return;
    try {
        await navigator.clipboard.writeText(text);
    } catch (_) {
        const textarea = document.createElement("textarea");
        textarea.value = text;
        textarea.setAttribute("readonly", "");
        textarea.style.position = "fixed";
        textarea.style.opacity = "0";
        document.body.appendChild(textarea);
        textarea.select();
        const copied = document.execCommand("copy");
        textarea.remove();
        if (!copied) throw new Error("Clipboard is unavailable");
    }
    showToast(message);
}

function showToast(message) {
    window.clearTimeout(state.toastTimer);
    elements.toast.textContent = message;
    elements.toast.classList.add("show");
    state.toastTimer = window.setTimeout(() => elements.toast.classList.remove("show"), 1800);
}

function statusRow(label, value) {
    const row = document.createElement("div");
    row.className = "status-row";
    const key = document.createElement("span");
    key.textContent = label;
    const val = document.createElement("strong");
    val.textContent = value;
    row.append(key, val);
    return row;
}

function renderVaultStatus(data) {
    const status = ["healthy", "standby", "unhealthy"].includes(data.status) ? data.status : "unhealthy";
    const labels = { healthy: "Vault online", standby: "Vault standby", unhealthy: "Vault unavailable" };
    elements.vaultStatusDot.className = `status-dot ${status}`;
    elements.vaultStatusLabel.textContent = labels[status];
    const title = document.createElement("div");
    title.className = "status-popover-title";
    title.textContent = "Vault backend";
    const rows = [
        statusRow("Status", labels[status].replace("Vault ", "")),
        statusRow("Initialized", data.initialized ? "Yes" : "No"),
        statusRow("Sealed", data.sealed ? "Yes" : "No")
    ];
    if (data.vault_version) rows.push(statusRow("Version", data.vault_version));
    if (data.cluster_name) rows.push(statusRow("Cluster", data.cluster_name));
    elements.vaultStatusPanel.replaceChildren(title, ...rows);
}

async function fetchVaultHealth() {
    try {
        const response = await fetch("/api/health", { cache: "no-store", credentials: "same-origin" });
        const data = await response.json();
        renderVaultStatus(data);
    } catch (_) {
        renderVaultStatus({ status: "unhealthy", initialized: false, sealed: false });
    }
}

async function fetchAppInfo() {
    try {
        const response = await fetch("/api/version", { cache: "no-store", credentials: "same-origin" });
        if (!response.ok) return;
        const data = await response.json();
        if (data.version) {
            elements.appVersion.textContent = `v${data.version}`;
            elements.appVersion.title = `Vault Wrapper ${data.version}`;
        }
        const configuredMax = Number(data.max_payload_size || data.max_request_size);
        if (Number.isFinite(configuredMax) && configuredMax > 0) {
            state.maxSize = configuredMax;
            updateSizeMeter();
        }
        if (data.github_url) {
            elements.githubLink.href = data.github_url;
        }
    } catch (_) {
        // Version information is non-essential.
    }
}

elements.themeToggle.addEventListener("click", () => {
    const next = elements.body.classList.contains("dark-mode") ? "light" : "dark";
    applyTheme(next);
    try { localStorage.setItem("vault-wrapper-theme", next); } catch (_) { /* optional */ }
});

document.querySelectorAll("[role=tab]").forEach((button) => {
    button.addEventListener("click", () => selectTab(button.dataset.tab));
    button.addEventListener("keydown", (event) => {
        if (event.key !== "ArrowLeft" && event.key !== "ArrowRight") return;
        event.preventDefault();
        const next = button.id === "wrapTab" ? "unwrapPanel" : "wrapPanel";
        selectTab(next, true);
    });
});

elements.attachButton.addEventListener("click", () => elements.fileInput.click());
elements.fileInput.addEventListener("change", () => {
    queueFiles(elements.fileInput.files);
    elements.fileInput.value = "";
});

let dragDepth = 0;
elements.dropZone.addEventListener("dragenter", (event) => {
    event.preventDefault();
    dragDepth += 1;
    elements.dropZone.classList.add("dragover");
});
elements.dropZone.addEventListener("dragover", (event) => {
    event.preventDefault();
    if (event.dataTransfer) event.dataTransfer.dropEffect = "copy";
});
elements.dropZone.addEventListener("dragleave", (event) => {
    event.preventDefault();
    dragDepth = Math.max(0, dragDepth - 1);
    if (dragDepth === 0) elements.dropZone.classList.remove("dragover");
});
elements.dropZone.addEventListener("drop", (event) => {
    event.preventDefault();
    dragDepth = 0;
    elements.dropZone.classList.remove("dragover");
    queueFiles(event.dataTransfer?.files);
});

wrapEditor.on("change", updateSizeMeter);
wrapEditor.on("drop", (_, event) => {
    event.preventDefault();
    event.stopPropagation();
    queueFiles(event.dataTransfer?.files);
});

elements.ttlPresets.addEventListener("click", (event) => {
    const button = event.target.closest("button[data-seconds]");
    if (!button) return;
    elements.ttl.value = button.dataset.seconds;
    elements.ttlPresets.querySelectorAll("button").forEach((preset) => preset.classList.toggle("active", preset === button));
    updateTtlPreview();
});
elements.ttl.addEventListener("input", () => {
    elements.ttlPresets.querySelectorAll("button").forEach((preset) => preset.classList.toggle("active", preset.dataset.seconds === elements.ttl.value));
    updateTtlPreview();
});

elements.wrapButton.addEventListener("click", createSecret);
elements.unwrapButton.addEventListener("click", openSecret);
elements.pasteButton.addEventListener("click", async () => {
    try {
        elements.unwrapInput.value = (await navigator.clipboard.readText()).trim();
        elements.unwrapInput.focus();
    } catch (_) {
        showAlert(elements.unwrapError, "Clipboard access is blocked. Paste the token into the field manually.");
    }
});
elements.toggleTokenVisibility.addEventListener("click", () => {
    const show = elements.unwrapInput.type === "password";
    elements.unwrapInput.type = show ? "text" : "password";
    elements.toggleTokenVisibility.setAttribute("aria-label", show ? "Hide token" : "Show token");
    elements.toggleTokenVisibility.title = show ? "Hide token" : "Show token";
});
elements.copyUnwrappedText.addEventListener("click", () => copyText(unwrapEditor.getValue(), "Text copied"));
elements.openAnotherButton.addEventListener("click", () => resetOpenedSecret());

document.addEventListener("click", (event) => {
    if (elements.vaultStatus.open && !elements.vaultStatus.contains(event.target)) elements.vaultStatus.open = false;
});
window.addEventListener("beforeunload", () => {
    revokeObjectUrls();
    state.receiptSource?.close();
});

function loadSharedToken() {
    const incomingToken = sharedTokenFromUrl();
    if (!incomingToken) return;
    resetOpenedSecret(false);
    elements.unwrapInput.value = incomingToken;
    elements.sharedTokenNotice.hidden = false;
    selectTab("unwrapPanel");
    setTimeout(() => elements.unwrapButton.focus(), 0);
}

window.addEventListener("hashchange", loadSharedToken);
loadSharedToken();
updateSizeMeter();
updateTtlPreview();
fetchAppInfo();
fetchVaultHealth();
window.setInterval(fetchVaultHealth, 30000);
window.setInterval(updateShareCountdowns, 1000);
