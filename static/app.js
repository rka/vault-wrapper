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

// Get elements
const dropZone = document.getElementById('dropZone');
const fileUploadIcon = document.getElementById('fileUploadIcon');
const wrapSuccess = document.getElementById('wrapSuccess');
const unwrapSuccess = document.getElementById('unwrapSuccess');

// Variable to hold uploaded file data
let uploadedFile = null;

// Handle drag over
dropZone.addEventListener('dragover', (event) => {
    event.preventDefault();
    dropZone.classList.add('dragover');
});

// Handle drag leave
dropZone.addEventListener('dragleave', () => {
    dropZone.classList.remove('dragover');
});

// Handle drop
dropZone.addEventListener('drop', (event) => {
    event.preventDefault();
    dropZone.classList.remove('dragover');
    const files = event.dataTransfer.files;
    if (files.length > 0) {
        handleFileUpload(files[0]);
    }
});

// Handle file upload icon click
fileUploadIcon.addEventListener('click', () => {
    const fileInput = document.createElement('input');
    fileInput.type = 'file';
    fileInput.onchange = () => {
        if (fileInput.files.length > 0) {
            handleFileUpload(fileInput.files[0]);
        }
    };
    fileInput.click();
});

// Handle file upload
function handleFileUpload(file) {
    const maxSize = 5 * 1024 * 1024; // 5 MB limit
    if (file.size > maxSize) {
        alert('File size exceeds 5 MB limit.');
        return;
    }

    const reader = new FileReader();
    reader.onload = function(e) {
        const base64Data = e.target.result.split(',')[1]; // Remove data URL prefix
        uploadedFile = {
            isFile: true,
            name: file.name,
            type: file.type,
            data: base64Data
        };
        // Inform the user that a file is ready
        wrapSuccess.textContent = `File "${file.name}" is ready to be wrapped.`;
        wrapSuccess.style.display = 'block';
        setTimeout(() => {
            wrapSuccess.style.display = 'none';
        }, 3000);
    };
    reader.readAsDataURL(file);
}

async function wrapData() {
    const inputText = wrapEditor.getValue();
    const ttl = document.getElementById('ttl').value;
    const detailsDiv = document.getElementById('wrapDetails');

    let dataObj = {
        text: inputText
    };

    if (uploadedFile) {
        dataObj.file = uploadedFile;
    }

    try {
        const response = await fetch('/wrap', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({ data: dataObj, ttl: ttl })
        });

        if (!response.ok) {
            throw new Error('Wrap request failed');
        }

        const data = await response.json();
        document.getElementById('wrappedToken').value = data.token;
        const url = new URL(window.location.href);
        url.searchParams.set('token', data.token);
        document.getElementById('wrappedLink').value = url.toString();
        detailsDiv.innerHTML = `<pre>${JSON.stringify(data.details, null, 2)}</pre>`;
        wrapSuccess.textContent = 'Data wrapped successfully!';
        wrapSuccess.style.display = 'block';
        setTimeout(() => {
            wrapSuccess.style.display = 'none';
        }, 3000);
        // Reset uploaded file after wrapping
        uploadedFile = null;
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

        // Clear previous content
        resultEditor.setValue('');
        const resultContainer = resultEditor.getWrapperElement();
        resultContainer.innerHTML = ''; // Clear previous results
        resultContainer.appendChild(resultEditor.getScrollerElement());

        if (data.file) {
            // Create a file bubble
            const fileBubble = document.createElement('div');
            fileBubble.className = 'file-bubble';
            fileBubble.textContent = `📄 ${data.file.name}`;
            fileBubble.title = 'Click to download';

            // Reconstruct file from Base64 data
            const blob = base64ToBlob(data.file.data, data.file.type);
            const url = URL.createObjectURL(blob);

            // Attach click event to download the file
            fileBubble.addEventListener('click', () => {
                const downloadLink = document.createElement('a');
                downloadLink.href = url;
                downloadLink.download = data.file.name;
                document.body.appendChild(downloadLink);
                downloadLink.click();
                document.body.removeChild(downloadLink);
                URL.revokeObjectURL(url); // Clean up the URL object
            });

            resultContainer.appendChild(fileBubble);
        }

        if (data.text) {
            // Display text data
            resultEditor.setValue(data.text);
        }

        unwrapSuccess.textContent = 'Data unwrapped successfully!';
        unwrapSuccess.style.display = 'block';
        setTimeout(() => {
            unwrapSuccess.style.display = 'none';
        }, 3000);
    } catch (error) {
        resultEditor.setValue(`Error: ${error.message}`);
    }
}


// Helper function to convert Base64 to Blob
function base64ToBlob(base64, type) {
    const binary = atob(base64);
    const array = [];
    for (let i = 0; i < binary.length; i++) {
        array.push(binary.charCodeAt(i));
    }
    return new Blob([new Uint8Array(array)], { type: type });
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

    // Save preference in cookie
    const mode = document.body.classList.contains('night-mode') ? 'dark' : 'light';
    document.cookie = `theme=${mode};path=/;max-age=31536000`; // Expires in 1 year
}

// On page load, check if a token is in the URL and unwrap it
window.onload = function() {
    // Check for theme preference in cookies
    const cookies = document.cookie.split(';').reduce((accumulator, cookie) => {
        const [key, value] = cookie.trim().split('=');
        accumulator[key] = value;
        return accumulator;
    }, {});

    if (cookies.theme === 'dark') {
        document.body.classList.add('night-mode');
        document.body.classList.remove('light-mode');
        wrapEditor.setOption('theme', 'material-darker');
        unwrapResultEditor.setOption('theme', 'material-darker');
    } else {
        document.body.classList.add('light-mode');
        document.body.classList.remove('night-mode');
        wrapEditor.setOption('theme', 'default');
        unwrapResultEditor.setOption('theme', 'default');
    }

    const urlParams = new URLSearchParams(window.location.search);
    const token = urlParams.get('token');
    if (token) {
        document.getElementById('unwrapInput').value = token;
        unwrapData(token);
    }
};
