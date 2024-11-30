const vault = require('node-vault')({
    apiVersion: 'v1',
    endpoint: 'http://vault:8200',
    token: 'root'
  });
  
  async function wrapData(data) {
    const { wrap_info } = await vault.write('sys/wrapping/wrap', { data });
    return wrap_info.token;
  }
  
  async function unwrapData(token) {
    const { data } = await vault.unwrap({ token });
    return data;
  }
  
  module.exports = { wrapData, unwrapData };