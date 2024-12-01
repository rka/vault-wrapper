// Initialize CodeMirror editors
let wrapEditor = CodeMirror(document.getElementById('wrapInput'), {
    lineNumbers: true,
    mode: 'javascript',
    theme: 'default',
    lineWrapping: true
});

let unwrapResultEditor = CodeMirror(document.getElementById('unwrapResult'), {
    lineNumbers: true,
    mode: 'javascript',
    theme: 'default',
    readOnly: true,
    lineWrapping: true
});

async function wrapData() {
    const input = wrapEditor.getValue();
    const ttl = document.getElementById('ttl').value;
    const detailsDiv = document.getElementById('wrapDetails');

    try {
        const response = await fetch('/wrap', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({ text: input, ttl: ttl })
        });

        if (!response.ok) {
            throw new Error('Wrap request failed');
        }

        const data = await response.json();
        const wrappedTokenElement = document.getElementById('wrappedToken');
        const wrappedLinkElement = document.getElementById('wrappedLink');
        if (wrappedTokenElement && wrappedLinkElement) {
            wrappedTokenElement.value = data.token;
            const url = new URL(window.location.href);
            url.searchParams.set('token', data.token);
            wrappedLinkElement.value = url.toString();
        } else {
            console.error('Wrapped token or link element not found');
        }
        detailsDiv.innerHTML = `<pre>${JSON.stringify(data.details, null, 2)}</pre>`;
    } catch (error) {
        detailsDiv.textContent = `Error: ${error.message}`;
    }
}

async function unwrapData(token) {
    const input = token || document.getElementById('unwrapInput').value;
    const resultEditor = unwrapResultEditor;

    try {
        const response = await fetch('/unwrap', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({ token: input })
        });

        if (!response.ok) {
            throw new Error('Unwrap request failed');
        }

        const data = await response.json();
        resultEditor.setValue(data.data);
    } catch (error) {
        resultEditor.setValue(`Error: ${error.message}`);
    }
}

function copyToClipboard(elementId) {
    const element = document.getElementById(elementId);
    const text = element.value;
    navigator.clipboard.writeText(text).catch(err => {
        console.error('Failed to copy text: ', err);
    });
}

function toggleNightMode() {
    document.body.classList.toggle('night-mode');
    document.body.classList.toggle('light-mode');
    const theme = document.body.classList.contains('night-mode') ? 'material-darker' : 'default';
    wrapEditor.setOption('theme', theme);
    unwrapResultEditor.setOption('theme', theme);
}

// On page load, check if a token is in the URL and unwrap it
window.onload = function() {
    // Set initial mode classes
    if (document.body.classList.contains('night-mode')) {
        document.body.classList.remove('light-mode');
    } else {
        document.body.classList.add('light-mode');
    }

    const urlParams = new URLSearchParams(window.location.search);
    const token = urlParams.get('token');
    if (token) {
        document.getElementById('unwrapInput').value = token;
        unwrapData(token);
    }
};
