from flask import Flask, request, render_template, redirect, url_for
import requests

app = Flask(__name__)

# Vault configuration
VAULT_ADDR = "http://vault:8200"
VAULT_TOKEN = "root"

@app.route("/", methods=["GET", "POST"])
def wrap_text():
    if request.method == "POST":
        text = request.form.get("text")
        if not text:
            return render_template("index.html", error="Please provide some text to wrap.")
        
        # Wrap the text using Vault API
        headers = {"X-Vault-Token": VAULT_TOKEN}
        data = {"value": text}
        response = requests.post(f"{VAULT_ADDR}/v1/sys/wrapping/wrap", headers=headers, json=data)
        
        if response.status_code == 200:
            token = response.json().get("wrap_info", {}).get("token")
            return render_template("index.html", wrapped_token=token)
        else:
            return render_template("index.html", error="Failed to wrap text. Check Vault logs.")
    
    return render_template("index.html")


@app.route("/unwrap", methods=["POST"])
def unwrap_token():
    token = request.form.get("token")
    if not token:
        return render_template("index.html", error="Please provide a token to unwrap.")
    
    # Unwrap the token using Vault API
    headers = {"X-Vault-Token": token}
    response = requests.post(f"{VAULT_ADDR}/v1/sys/wrapping/unwrap", headers=headers)
    
    if response.status_code == 200:
        unwrapped_data = response.json().get("data", {}).get("value")
        return render_template("index.html", unwrapped_data=unwrapped_data)
    else:
        return render_template("index.html", error="Failed to unwrap token. Check the provided token.")

if __name__ == "__main__":
    app.run(host="0.0.0.0", port=5000, debug=True)
