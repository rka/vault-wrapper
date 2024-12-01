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
        resultDiv.innerHTML = `Wrapped Token: ${data.token}`;
    } catch (error) {
        resultDiv.textContent = `Error: ${error.message}`;
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

// Highlight input as user types
document.getElementById('wrapInput').addEventListener('input', function() {
    const wrappedText = wrapInCodeTags(this.value);
    document.getElementById('wrapResult').innerHTML = wrappedText;
    highlightCode(document.getElementById('wrapResult'));
});

// Initial highlight
highlightCode(document);