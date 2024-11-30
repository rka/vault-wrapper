from flask import Flask, request, render_template, redirect, url_for
import requests
import logging

# Set up logging
logging.basicConfig(level=logging.DEBUG,
                    format='%(asctime)s - %(levelname)s - %(message)s')
logger = logging.getLogger(__name__)

app = Flask(__name__)

# Vault configuration
VAULT_ADDR = "http://vault:8200"
VAULT_TOKEN = "root"

@app.route("/", methods=["GET", "POST"])
def wrap_text():
    if request.method == "POST":
        text = request.form.get("text")
        if not text:
            logger.warning("No text provided to wrap.")
            return render_template("index.html", error="Please provide some text to wrap.")
        
        # Log the request details
        logger.debug(f"Received text to wrap: {text}")
        
        # Wrap the text using Vault API
        headers = {"X-Vault-Token": VAULT_TOKEN}
        data = {"value": text}
        try:
            response = requests.post(f"{VAULT_ADDR}/v1/sys/wrapping/wrap", headers=headers, json=data)
            logger.debug(f"Vault response: {response.status_code}, {response.text}")
            
            if response.status_code == 200:
                token = response.json().get("wrap_info", {}).get("token")
                logger.info(f"Successfully wrapped the text. Token: {token}")
                return render_template("index.html", wrapped_token=token)
            else:
                logger.error(f"Failed to wrap text. Response: {response.status_code}, {response.text}")
                return render_template("index.html", error="Failed to wrap text. Check Vault logs.")
        except Exception as e:
            logger.error(f"Error communicating with Vault: {e}")
            return render_template("index.html", error="Failed to wrap text. Check Vault logs.")
    
    return render_template("index.html")


@app.route("/unwrap", methods=["POST"])
def unwrap_token():
    token = request.form.get("token")
    if not token:
        logger.warning("No token provided to unwrap.")
        return render_template("index.html", error="Please provide a token to unwrap.")
    
    # Log the request details
    logger.debug(f"Received token to unwrap: {token}")
    
    # Unwrap the token using Vault API
    headers = {"X-Vault-Token": token}
    try:
        response = requests.post(f"{VAULT_ADDR}/v1/sys/wrapping/unwrap", headers=headers)
        logger.debug(f"Vault response: {response.status_code}, {response.text}")
        
        if response.status_code == 200:
            unwrapped_data = response.json().get("data", {}).get("value")
            logger.info(f"Successfully unwrapped the token. Data: {unwrapped_data}")
            return render_template("index.html", unwrapped_data=unwrapped_data)
        else:
            logger.error(f"Failed to unwrap token. Response: {response.status_code}, {response.text}")
            return render_template("index.html", error="Failed to unwrap token. Check the provided token.")
    except Exception as e:
        logger.error(f"Error communicating with Vault: {e}")
        return render_template("index.html", error="Failed to unwrap token. Check the provided token.")

if __name__ == "__main__":
    app.run(host="0.0.0.0", port=5000, debug=True)
