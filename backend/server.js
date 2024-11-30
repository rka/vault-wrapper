const express = require('express');
const path = require('path');
const cors = require('cors');
const vaultService = require('./vaultService');

const app = express();
app.use(cors());
app.use(express.json());

// Serve static files from the React app
app.use(express.static(path.join(__dirname, 'public')));

app.post('/api/wrap', async (req, res) => {
  try {
    const { data } = req.body;
    const token = await vaultService.wrapData(data);
    res.json({ token });
  } catch (error) {
    res.status(500).json({ error: error.message });
  }
});

app.post('/api/unwrap', async (req, res) => {
  try {
    const { token } = req.body;
    const data = await vaultService.unwrapData(token);
    res.json({ data });
  } catch (error) {
    res.status(500).json({ error: error.message });
  }
});

// Handle any requests that don't match the ones above
app.get('*', (req, res) => {
  res.sendFile(path.join(__dirname, 'public', 'index.html'));
});

const PORT = process.env.PORT || 3001;
app.listen(PORT, () => console.log(`Server running on port ${PORT}`));