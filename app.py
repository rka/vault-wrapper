import logging
from flask import Flask, request, render_template
import requests

app = Flask(__name__)

# Vault configuration
VAULT_ADDR = "http://vault:8200"
VAULT_TOKEN = "root"

# Set up Flask's default logger to log at DEBUG level
app.logger.setLevel(logging.DEBUG)

# Create a console handler to display logs on the terminal
ch = logging.StreamHandler()
ch.setLevel(logging.DEBUG)

# Create a formatter for the log messages
formatter = logging.Formatter('%(asctime)s - %(levelname)s - %(message)s')
ch.setFormatter(formatter)

# Add the handler to Flask's default logger
app.logger.addHandler(ch)

@app.route("/", methods=["GET", "POST"])
def wrap_text():
    if request.method == "POST":
        text = request.form.get("text")
        if not text:
            app.logger.warning("No text provided to wrap.")
            return render_template("index.html", error="Please provide some text to wrap.")
        
        # Log the request details
        app.logger.debug(f"Received text to wrap: {text}")
        
        # Wrap the text using Vault API
        headers = {"X-Vault-Token": VAULT_TOKEN}
        data = {"value": text}
        try:
            response = requests.post(f"{VAULT_ADDR}/v1/sys/wrapping/wrap", headers=headers, json=data)
            app.logger.debug(f"Vault response: {response.status_code}, {response.text}")
            
            if response.status_code == 200:
                token = response.json().get("wrap_info", {}).get("token")
                app.logger.info(f"Successfully wrapped the text. Token: {token}")
                return render_template("index.html", wrapped_token=token)
            else:
                app.logger.error(f"Failed to wrap text. Response: {response.status_code}, {response.text}")
                return render_template("index.html", error="Failed to wrap text. Check Vault logs.")
        except Exception as e:
            app.logger.error(f"Error communicating with Vault: {e}")
            return render_template("index.html", error="Failed to wrap text. Check Vault logs.")
    
    return render_template("index.html")


@app.route("/unwrap", methods=["POST"])
def unwrap_token():
    token = request.form.get("token")
    if not token:
        app.logger.warning("No token provided to unwrap.")
        return render_template("index.html", error="Please provide a token to unwrap.")
    
    # Log the request details
    app.logger.debug(f"Received token to unwrap: {token}")
    
    # Unwrap the token using Vault API
    headers = {"X-Vault-Token": token}
    try:
        response = requests.post(f"{VAULT_ADDR}/v1/sys/wrapping/unwrap", headers=headers)
        app.logger.debug(f"Vault response: {response.status_code}, {response.text}")
        
        if response.status_code == 200:
            unwrapped_data = response.json().get("data", {}).get("value")
            app.logger.info(f"Successfully unwrapped the token. Data: {unwrapped_data}")
            return render_template("index.html", unwrapped_data=unwrapped_data)
        else:
            app.logger.error(f"Failed to unwrap token. Response: {response.status_code}, {response.text}")
            return render_template("index.html", error="Failed to unwrap token. Check the provided token.")
    except Exception as e:
        app.logger.error(f"Error communicating with Vault: {e}")
        return render_template("index.html", error="Failed to unwrap token. Check the provided token.")

if __name__ == "__main__":
    app.run(host="0.0.0.0", port=5000, debug=True)
