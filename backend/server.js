const express = require('express');
const cors = require('cors');
const vaultService = require('./vaultService');

const app = express();
app.use(cors());
app.use(express.json());

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

const PORT = process.env.PORT || 3001;
app.listen(PORT, () => console.log(`Server running on port ${PORT}`));