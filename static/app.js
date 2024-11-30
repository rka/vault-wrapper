async function wrapData() {
    const input = document.getElementById('wrapInput').value;
    const resultDiv = document.getElementById('wrapResult');

    try {
        const response = await fetch('/wrap', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({ text: input })
        });

        if (!response.ok) {
            throw new Error('Wrap request failed');
        }

        const data = await response.json();
        resultDiv.textContent = `Wrapped Token: ${data.token}`;
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
        resultDiv.textContent = `Unwrapped Data: ${data.data}`;
    } catch (error) {
        resultDiv.textContent = `Error: ${error.message}`;
    }
}