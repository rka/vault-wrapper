const express = require('express');
const bodyParser = require('body-parser');
const axios = require('axios');
require('dotenv').config();

const app = express();
const port = 3000;

// Middleware
app.use(bodyParser.json());

// Vault configuration
const vaultUrl = process.env.VAULT_ADDR || 'http://vault:8200';
const roleId = process.env.ROLE_ID;
const secretId = process.env.SECRET_ID;

let clientToken = null;

// Authenticate with Vault using AppRolee
async function authenticateWithVault() {
  try {
    const response = await axios.post(`${vaultUrl}/v1/auth/approle/login`, {
      role_id: roleId,
      secret_id: secretId,
    });
    clientToken = response.data.auth.client_token;
    console.log('Authenticated with Vault');
  } catch (err) {
    console.error('Error authenticating with Vault:', err.message);
  }
}

// Endpoint to wrap data
app.post('/wrap', async (req, res) => {
  try {
    const data = req.body.data;
    const response = await axios.post(
      `${vaultUrl}/v1/sys/wrapping/wrap`,
      { data },
      {
        headers: {
          'X-Vault-Token': clientToken,
        },
      }
    );
    res.json({ wrapToken: response.data.wrap_info.token });
  } catch (err) {
    console.error('Error wrapping data:', err.message);
    res.status(500).send('Failed to wrap data');
  }
});

// Endpoint to unwrap data
app.post('/unwrap', async (req, res) => {
  try {
    const wrapToken = req.body.wrapToken;
    const response = await axios.post(
      `${vaultUrl}/v1/sys/wrapping/unwrap`,
      {},
      {
        headers: {
          'X-Vault-Token': wrapToken,
        },
      }
    );
    res.json(response.data.data);
  } catch (err) {
    console.error('Error unwrapping data:', err.message);
    res.status(500).send('Failed to unwrap data');
  }
});

// Start the server and authenticate with Vault
app.listen(port, async () => {
  console.log(`Backend listening at http://localhost:${port}`);
  await authenticateWithVault();
});
