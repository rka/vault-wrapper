import React, { useState } from 'react';
import './App.css';

function App() {
  const [mode, setMode] = useState('wrap');
  const [input, setInput] = useState('');
  const [result, setResult] = useState('');

  const handleSubmit = async (e) => {
    e.preventDefault();
    const endpoint = mode === 'wrap' ? '/wrap' : '/unwrap';
    const body =
      mode === 'wrap' ? { data: input } : { wrapToken: input };

    try {
      const response = await fetch(endpoint, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(body),
      });
      const data = await response.json();
      setResult(JSON.stringify(data, null, 2));
    } catch (err) {
      setResult(`Error: ${err.message}`);
    }
  };

  return (
    <div className="App">
      <h1>Vault Wrapper App</h1>
      <form onSubmit={handleSubmit}>
        <div>
          <label>
            Mode:
            <select value={mode} onChange={(e) => setMode(e.target.value)}>
              <option value="wrap">Wrap</option>
              <option value="unwrap">Unwrap</option>
            </select>
          </label>
        </div>
        <div>
          <label>
            Input:
            <textarea
              value={input}
              onChange={(e) => setInput(e.target.value)}
              rows="5"
            />
          </label>
        </div>
        <button type="submit">Submit</button>
      </form>
      <h2>Result</h2>
      <pre>{result}</pre>
    </div>
  );
}

export default App;
