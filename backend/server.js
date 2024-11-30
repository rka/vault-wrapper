const express = require('express');
const path = require('path');
const cors = require('cors');
const vaultService = require('./vaultService');
const fs = require('fs');

const app = express();
app.use(cors());
app.use(express.json());

// Debugging middleware
app.use((req, res, next) => {
  console.log(`Received ${req.method} request to ${req.path}`);
  next();
});

// Check public directory contents
console.log('Public directory contents:');
try {
  const files = fs.readdirSync(path.join(__dirname, 'public'));
  console.log(files);
} catch (error) {
  console.error('Error reading public directory:', error);
}

// Serve static files from the React app
app.use(express.static(path.join(__dirname, 'public'), {
  setHeaders: (res, path) => {
    console.log('Serving file:', path);
  }
}));

// API Routes
app.post('/api/wrap', async (req, res) => {
  try {
    const { data } = req.body;
    const token = await vaultService.wrapData(data);
    res.json({ token });
  } catch (error) {
    console.error('Wrap error:', error);
    res.status(500).json({ error: error.message });
  }
});

app.post('/api/unwrap', async (req, res) => {
  try {
    const { token } = req.body;
    const data = await vaultService.unwrapData(token);
    res.json({ data });
  } catch (error) {
    console.error('Unwrap error:', error);
    res.status(500).json({ error: error.message });
  }
});

// Catch-all route with extensive logging
app.get('*', (req, res) => {
  console.log('Catch-all route hit');
  console.log('Requested path:', req.path);
  
  const indexPath = path.join(__dirname, 'public', 'index.html');
  console.log('Attempting to serve index.html from:', indexPath);
  
  try {
    if (fs.existsSync(indexPath)) {
      res.sendFile(indexPath);
    } else {
      console.error('index.html not found');
      res.status(404).send('index.html not found');
    }
  } catch (error) {
    console.error('Error serving index.html:', error);
    res.status(500).send('Internal server error');
  }
});

const PORT = process.env.PORT || 3001;
app.listen(PORT, () => {
  console.log(`Server running on port ${PORT}`);
  console.log('Current directory:', __dirname);
});