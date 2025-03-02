// Initialize CodeMirror editors with drop prevention
let wrapEditor = CodeMirror(document.getElementById('wrapInput'), {
    lineNumbers: true,
    mode: 'javascript',
    theme: 'dracula',
    lineWrapping: true,
    dragDrop: false
});

let unwrapResultEditor = CodeMirror(document.getElementById('unwrapResult'), {
    lineNumbers: true,
    mode: 'javascript',
    theme: 'dracula',
    readOnly: true,
    lineWrapping: true
});

// Core elements
const dropZone = document.getElementById('dropZone');
const fileUploadIcon = document.getElementById('fileUploadIcon');
const wrapSuccess = document.getElementById('wrapSuccess');
const unwrapSuccess = document.getElementById('unwrapSuccess');
const maxSize = 5 * 1024 * 1024;
let uploadedFiles = [];

// Handle drag over
dropZone.addEventListener('dragover', (event) => {
    event.preventDefault();
    event.dataTransfer.dropEffect = 'copy';
    dropZone.classList.add('dragover');
});

// Prevent default drop behavior that would read text files as text
dropZone.addEventListener('drop', (event) => {
    event.preventDefault();
    event.stopPropagation();
    dropZone.classList.remove('dragover');
    
    // Handle only as files, never as text
    const files = event.dataTransfer.files;
    if (files.length > 0) {
        handleFileUpload(files);
    }
    
    // Clear any text data that might have been interpreted
    event.dataTransfer.clearData();
});

// Handle drag leave
dropZone.addEventListener('dragleave', () => {
    dropZone.classList.remove('dragover');
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

// Prevent drop on the CodeMirror element explicitly
wrapEditor.on('drop', (cm, event) => {
    event.preventDefault();
    event.stopPropagation();
    handleFileUpload(event.dataTransfer.files);
});

// Prevent dragover on CodeMirror
wrapEditor.on('dragover', (cm, event) => {
    event.preventDefault();
    event.stopPropagation();
    dropZone.classList.add('dragover');
});

// Handle drag leave
wrapEditor.on('dragleave', (cm, event) => {
    event.preventDefault();
    event.stopPropagation();
    dropZone.classList.remove('dragover');
});

// Handle file uploads (multiple files)
function handleFileUpload(files) {
    const totalSize = Array.from(files).reduce((sum, file) => sum + file.size, 0);
    const currentSize = uploadedFiles.reduce((sum, file) => sum + file.size, 0);
    
    if (totalSize + currentSize > maxSize) {
        alert(`Adding these files would exceed the 5 MB limit.`);
        return;
    }

    for (const file of files) {
        if (file.size > maxSize) {
            alert(`File "${file.name}" exceeds 5 MB limit and will not be added.`);
            continue;
        }

        // Always treat as file, regardless of type
        const reader = new FileReader();
        reader.onload = function(e) {
            const base64Data = e.target.result.split(',')[1];
            const uploadedFile = {
                isFile: true,
                name: file.name,
                type: file.type || 'text/plain', // Fallback type for unknown files
                data: base64Data,
                size: file.size
            };
            uploadedFiles.push(uploadedFile);
            displayWrapFileBubbles();
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
    const percentage = (totalSize / maxSize) * 100;
    const cappedPercentage = Math.min(percentage, 100);

    const sizeBar = document.getElementById('sizeBar');
      const sizeBarInner = document.getElementById('sizeBarInner');
    sizeBarInner.style.width = `${cappedPercentage}%`;
    
    const totalSizeFormatted = formatFileSize(totalSize);
    const maxSizeFormatted = formatFileSize(maxSize);
    sizeBar.setAttribute('data-size', `${totalSizeFormatted} / ${maxSizeFormatted}`);

    sizeBarInner.classList.remove('warning', 'error');
    if (percentage > 90) {
        sizeBarInner.classList.add('error');
    } else if (percentage > 75) {
        sizeBarInner.classList.add('warning');
    }
}

// Update size bar when text changes
wrapEditor.on('change', updateSizeBar);

async function wrapData() {
    const wrapButton = document.querySelector('.wrap-section button');
    const errorDiv = document.getElementById('wrapError');
    wrapButton.classList.add('loading');
    
    // Clear previous error
    errorDiv.style.display = 'none';

    const inputText = wrapEditor.getValue();
    const textSize = getTextSizeInBytes(inputText);
    const filesSize = uploadedFiles.reduce((total, file) => total + file.size, 0);
    const totalSize = textSize + filesSize;

    if (totalSize > maxSize) {
        errorDiv.textContent = 'Total size exceeds the maximum allowed size of 5 MB.';
        errorDiv.style.display = 'block';
        wrapButton.classList.remove('loading');
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
        errorDiv.textContent = 'Please enter text or upload files to wrap.';
        errorDiv.style.display = 'block';
        wrapButton.classList.remove('loading');
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
        
        // Show token warning only after successful wrap
        document.querySelector('.token-warning').style.display = 'flex';
        
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
        errorDiv.textContent = `Error: ${error.message}`;
        errorDiv.style.display = 'block';
        detailsDiv.textContent = '';
    } finally {
        wrapButton.classList.remove('loading');
    }
}

async function unwrapData(token) {
    const unwrapButton = document.querySelector('.unwrap-section button');
    const resultEditor = unwrapResultEditor;
    const errorDiv = document.getElementById('unwrapError');
    unwrapButton.classList.add('loading');
    
    // Clear previous states
    errorDiv.style.display = 'none';
    resultEditor.setValue('');
    resultEditor.getWrapperElement().style.display = 'none';
    
    try {
        const tokenToUnwrap = token || document.getElementById('unwrapInput').value;
        const response = await fetch('/unwrap', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({ token: tokenToUnwrap })
        });

        if (!response.ok) {
            const errorText = await response.text();
            if (response.status === 404) {
                errorDiv.textContent = 'This token has already been used or does not exist. Tokens can only be unwrapped once and are permanently deleted after unwrapping.';
            } else {
                errorDiv.textContent = 'Error: ' + errorText;
            }
            errorDiv.style.display = 'block';
            return;
        }

        const data = await response.json();
        let contentAdded = false;
        const fileBubbleContainer = document.getElementById('fileBubbleContainer');
        fileBubbleContainer.innerHTML = '';

        // Display token info if available
        if (data.wrapping_info) {
            const tokenInfoDiv = document.createElement('div');
            tokenInfoDiv.className = 'token-info';
            tokenInfoDiv.innerHTML = `
                <h3>Token Information</h3>
                <pre>${JSON.stringify(data.wrapping_info, null, 2)}</pre>
            `;
            fileBubbleContainer.appendChild(tokenInfoDiv);
        }

        // Handle unwrapped data
        if (data.data) {
            if (data.data.files && Array.isArray(data.data.files)) {
                data.data.files.forEach(file => {
                    const blob = base64ToBlob(file.data, file.type);
                    const url = URL.createObjectURL(blob);

                    const fileBubble = document.createElement('div');
                    fileBubble.className = 'file-bubble';
                    const fileSizeFormatted = formatFileSize(file.size);
                    fileBubble.textContent = `📄 ${file.name} (${fileSizeFormatted})`;
                    fileBubble.title = 'Click to download';

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

            if (data.data.text) {
                resultEditor.setValue(data.data.text);
                resultEditor.getWrapperElement().style.display = 'block';
                // Force CodeMirror to refresh and render properly
                setTimeout(() => {
                    resultEditor.refresh();
                }, 1);
                contentAdded = true;
            }
        }

        if (!contentAdded) {
            errorDiv.textContent = 'No data found in the unwrapped content.';
            errorDiv.style.display = 'block';
        } else {
            unwrapSuccess.textContent = 'Data unwrapped successfully!';
            unwrapSuccess.style.display = 'block';
            setTimeout(() => {
                unwrapSuccess.style.display = 'none';
            }, 3000);
        }
    } catch (error) {
        console.error('Error during unwrapping:', error);
        errorDiv.textContent = 'An error occurred while unwrapping the data.';
        errorDiv.style.display = 'block';
    } finally {
        unwrapButton.classList.remove('loading');
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

// Add this after the existing window.onload function
window.addEventListener('DOMContentLoaded', () => {
    // Hide token warning initially
    document.querySelector('.token-warning').style.display = 'none';
});
