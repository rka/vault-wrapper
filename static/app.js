// Initialize CodeMirror editors
let wrapEditor = CodeMirror(document.getElementById('wrapInput'), {
    lineNumbers: true,
    mode: 'javascript',
    theme: 'dracula',
    lineWrapping: true
});

let unwrapResultEditor = CodeMirror(document.getElementById('unwrapResult'), {
    lineNumbers: true,
    mode: 'javascript',
    theme: 'dracula',
    readOnly: true,
    lineWrapping: true
});

// Get elements
const dropZone = document.getElementById('dropZone');
const fileUploadIcon = document.getElementById('fileUploadIcon');
const wrapSuccess = document.getElementById('wrapSuccess');
const unwrapSuccess = document.getElementById('unwrapSuccess');

// Variable to hold uploaded files data
let uploadedFiles = [];

// Maximum allowed size (5 MB)
const maxSize = 5 * 1024 * 1024; // 5 MB limit

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
        handleFileUpload(files);
    }
});

// Handle file upload icon click
fileUploadIcon.addEventListener('click', () => {
    const fileInput = document.createElement('input');
    fileInput.type = 'file';
    fileInput.multiple = true;
    fileInput.onchange = () => {
        if (fileInput.files.length > 0) {
            handleFileUpload(fileInput.files);
        }
    };
    fileInput.click();
});

// Handle file uploads (multiple files)
function handleFileUpload(files) {
    for (const file of files) {
        if (file.size > maxSize) {
            alert(`File "${file.name}" exceeds 5 MB limit and will not be added.`);
            continue;
        }

        const reader = new FileReader();
        reader.onload = function(e) {
            const base64Data = e.target.result.split(',')[1]; // Remove data URL prefix
            const uploadedFile = {
                isFile: true,
                name: file.name,
                type: file.type,
                data: base64Data,
                size: file.size // Include file size
            };
            uploadedFiles.push(uploadedFile);
            // Display the file bubbles in the wrap section
            displayWrapFileBubbles();
            // Update size bar
            updateSizeBar();
        };
        reader.readAsDataURL(file);
    }
}

// Display the file bubbles in the wrap section
function displayWrapFileBubbles() {
    const wrapFileBubbleContainer = document.getElementById('wrapFileBubbleContainer');
    wrapFileBubbleContainer.innerHTML = ''; // Clear previous content

    uploadedFiles.forEach((file, index) => {
        const fileBubble = document.createElement('div');
        fileBubble.className = 'file-bubble';
        const fileSizeFormatted = formatFileSize(file.size);
        fileBubble.textContent = `📄 ${file.name} (${fileSizeFormatted})`;
        fileBubble.title = 'Click to remove';

        // Attach click event to remove the file
        fileBubble.addEventListener('click', () => {
            uploadedFiles.splice(index, 1);
            displayWrapFileBubbles();
            updateSizeBar();
        });

        wrapFileBubbleContainer.appendChild(fileBubble);
    });
}

// Format file size into human-readable format
function formatFileSize(bytes) {
    const sizes = ['Bytes', 'KB', 'MB', 'GB', 'TB'];
    if (bytes === 0) return '0 Bytes';
    const i = parseInt(Math.floor(Math.log(bytes) / Math.log(1024)), 10);
    const size = bytes / Math.pow(1024, i);
    return `${size.toFixed(2)} ${sizes[i]}`;
}

// Get text size in bytes
function getTextSizeInBytes(text) {
    return new TextEncoder().encode(text).length;
}

// Update size bar
function updateSizeBar() {
    const inputText = wrapEditor.getValue();
    const textSize = getTextSizeInBytes(inputText);
    const filesSize = uploadedFiles.reduce((total, file) => total + file.size, 0);
    const totalSize = textSize + filesSize;

    // Get the size bar inner div
    const sizeBarInner = document.getElementById('sizeBarInner');

    const percentage = (totalSize / maxSize) * 100;
    const cappedPercentage = Math.min(percentage, 100);

    sizeBarInner.style.width = `${cappedPercentage}%`;

    // Change color if over limit
    if (totalSize > maxSize) {
        sizeBarInner.style.backgroundColor = 'red';
    } else {
        sizeBarInner.style.backgroundColor = 'var(--primary-color)';
    }

    // Display the size as text in the title
    const sizeBar = document.getElementById('sizeBar');
    sizeBar.title = `Total size: ${formatFileSize(totalSize)} / ${formatFileSize(maxSize)}. Maximum allowed is 5 MB.`;
}

// Update size bar when text changes
wrapEditor.on('change', updateSizeBar);

async function wrapData() {
    const inputText = wrapEditor.getValue();
    const textSize = getTextSizeInBytes(inputText);
    const filesSize = uploadedFiles.reduce((total, file) => total + file.size, 0);
    const totalSize = textSize + filesSize;

    if (totalSize > maxSize) {
        alert('Total size exceeds the maximum allowed size of 5 MB.');
        return;
    }

    const ttl = document.getElementById('ttl').value;
    const detailsDiv = document.getElementById('wrapDetails');

    let dataObj = {};

    // Include text only if it's not empty
    if (inputText.trim() !== '') {
        dataObj.text = inputText;
    }

    if (uploadedFiles.length > 0) {
        dataObj.files = uploadedFiles;
    }

    if (Object.keys(dataObj).length === 0) {
        alert('Please enter text or upload files to wrap.');
        return;
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
        // Reset uploaded files and clear file bubbles after wrapping
        uploadedFiles = [];
        document.getElementById('wrapFileBubbleContainer').innerHTML = '';
        // Update size bar
        updateSizeBar();
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
        console.log('Unwrapped data:', data);

        // Get containers
        const fileBubbleContainer = document.getElementById('fileBubbleContainer');

        // Clear previous content
        resultEditor.setValue('');
        fileBubbleContainer.innerHTML = '';

        let contentAdded = false;

        if (data.files && Array.isArray(data.files)) {
            console.log('Files data found:', data.files);
            data.files.forEach(file => {
                // Reconstruct file from Base64 data
                const blob = base64ToBlob(file.data, file.type);
                const url = URL.createObjectURL(blob);

                // Create file bubble
                const fileBubble = document.createElement('div');
                fileBubble.className = 'file-bubble';
                const fileSizeFormatted = formatFileSize(file.size);
                fileBubble.textContent = `📄 ${file.name} (${fileSizeFormatted})`;
                fileBubble.title = 'Click to download';

                // Attach click event
                fileBubble.addEventListener('click', () => {
                    const downloadLink = document.createElement('a');
                    downloadLink.href = url;
                    downloadLink.download = file.name;
                    document.body.appendChild(downloadLink);
                    downloadLink.click();
                    document.body.removeChild(downloadLink);
                    URL.revokeObjectURL(url);
                });

                fileBubbleContainer.appendChild(fileBubble);
                contentAdded = true;
            });
        }

        if (data.text) {
            console.log('Text data found:', data.text);
            // Display text data
            resultEditor.setValue(data.text);
            resultEditor.getWrapperElement().style.display = 'block';
            contentAdded = true;
        } else {
            // Hide the editor if no text is present
            resultEditor.getWrapperElement().style.display = 'none';
        }

        if (!contentAdded) {
            resultEditor.setValue('No data found.');
            resultEditor.getWrapperElement().style.display = 'block';
        }

        unwrapSuccess.textContent = 'Data unwrapped successfully!';
        unwrapSuccess.style.display = 'block';
        setTimeout(() => {
            unwrapSuccess.style.display = 'none';
        }, 3000);
    } catch (error) {
        console.error('Error during unwrapping:', error);
        resultEditor.setValue(`Error: ${error.message}`);
        resultEditor.getWrapperElement().style.display = 'block';
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
    navigator.clipboard.writeText(text).then(() => {
        showCopiedBanner();
    }).catch(err => {
        console.error('Failed to copy text: ', err);
    });
}

function showCopiedBanner() {
    const banner = document.getElementById('copiedBanner');
    banner.classList.add('show');
    setTimeout(() => {
        banner.classList.remove('show');
    }, 1500);
}

function toggleNightMode() {
    document.body.classList.toggle('night-mode');
    document.body.classList.toggle('light-mode');
    // Keep using dracula theme for both modes
    wrapEditor.setOption('theme', 'dracula');
    unwrapResultEditor.setOption('theme', 'dracula');

    // Save preference in cookie
    const mode = document.body.classList.contains('night-mode') ? 'dark' : 'light';
    document.cookie = `theme=${mode};path=/;max-age=31536000`;
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
        wrapEditor.setOption('theme', 'dracula');
        unwrapResultEditor.setOption('theme', 'dracula');
    } else {
        document.body.classList.add('light-mode');
        document.body.classList.remove('night-mode');
        wrapEditor.setOption('theme', 'dracula');
        unwrapResultEditor.setOption('theme', 'dracula');
    }

    const urlParams = new URLSearchParams(window.location.search);
    const token = urlParams.get('token');
    if (token) {
        document.getElementById('unwrapInput').value = token;
        unwrapData(token);
    }

    // Initialize size bar
    updateSizeBar();
};
