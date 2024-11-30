import React, { useState } from 'react';
import './App.css';

function App() {
  const [inputText, setInputText] = useState('');
  const [wrappedToken, setWrappedToken] = useState('');
  const [unwrappedText, setUnwrappedText] = useState('');

  const handleWrap = async () => {
    const response = await fetch('/api/wrap', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ data: inputText }),
    });
    const data = await response.json();
    setWrappedToken(data.token);
  };

  const handleUnwrap = async () => {
    const response = await fetch('/api/unwrap', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ token: wrappedToken }),
    });
    const data = await response.json();
    setUnwrappedText(data.data);
  };

  return (
    <div className="App">
      <header className="App-header">
        <h1>Vault Wrap/Unwrap</h1>
      </header>
      <main>
        <section>
          <h2>Wrap</h2>
          <textarea
            value={inputText}
            onChange={(e) => setInputText(e.target.value)}
            placeholder="Enter text or code to wrap"
          />
          <button onClick={handleWrap}>Wrap</button>
          {wrappedToken && (
            <div>
              <h3>Wrapped Token:</h3>
              <pre>{wrappedToken}</pre>
            </div>
          )}
        </section>
        <section>
          <h2>Unwrap</h2>
          <input
            type="text"
            value={wrappedToken}
            onChange={(e) => setWrappedToken(e.target.value)}
            placeholder="Enter wrapped token"
          />
          <button onClick={handleUnwrap}>Unwrap</button>
          {unwrappedText && (
            <div>
              <h3>Unwrapped Data:</h3>
              <pre>{unwrappedText}</pre>
            </div>
          )}
        </section>
      </main>
    </div>
  );
}

export default App;