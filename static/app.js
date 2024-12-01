function highlightCode(element) {
    element.querySelectorAll('pre code').forEach((block) => {
        hljs.highlightBlock(block);
    });
}

function wrapInCodeTags(text) {
    return `<pre><code>${text}</code></pre>`;
}

async function wrapData() {
    const input = document.getElementById('wrapInput').value;
    const ttl = document.getElementById('ttl').value;
    const resultDiv = document.getElementById('wrapResult');
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
        document.getElementById('wrappedToken').textContent = data.token;
        detailsDiv.innerHTML = `<pre>${JSON.stringify(data.details, null, 2)}</pre>`;
        highlightCode(detailsDiv);
    } catch (error) {
        resultDiv.textContent = `Error: ${error.message}`;
        detailsDiv.textContent = '';
    }
}

async function unwrapData() {
    const input = document.getElementById('unwrapInput').value;
    const resultDiv = document.getElementById('unwrapResult');

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
        resultDiv.innerHTML = wrapInCodeTags(data.data);
        highlightCode(resultDiv);
    } catch (error) {
        resultDiv.textContent = `Error: ${error.message}`;
    }
}

function copyToClipboard(elementId) {
    const element = document.getElementById(elementId);
    const text = element.textContent;
    navigator.clipboard.writeText(text).then(() => {
        alert('Token copied to clipboard!');
    }).catch(err => {
        console.error('Failed to copy text: ', err);
    });
}

// Highlight input as user types
document.getElementById('wrapInput').addEventListener('input', function() {
    const wrappedText = wrapInCodeTags(this.value);
    document.getElementById('wrapResult').innerHTML = wrappedText;
    highlightCode(document.getElementById('wrapResult'));
});

// Night mode toggle
document.getElementById('nightModeToggle').addEventListener('change', function() {
    document.body.classList.toggle('night-mode');
});

// Initial highlight
highlightCode(document);