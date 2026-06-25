#!/usr/bin/python3
import sys
import os
import json
import urllib.request
import traceback

# Optional: Enable traceback display in the browser (for debugging only)
import cgitb
cgitb.enable()

try:
    from cryptography.hazmat.primitives.ciphers.aead import AESGCM
except ImportError:
    print("Content-Type: application/json\n")
    print(json.dumps({"error": "The 'cryptography' module is missing on the server."}))
    sys.exit(1)

# --- Configuration ---
# This must be the exact same hex string as defined in your Golang client!
SHARED_KEY = bytes.fromhex("HIER_DEINEN_64_ZEICHEN_HEX_KEY_EINTRAGEN")

def encrypt_payload(payload_bytes: bytes, key: bytes) -> tuple[bytes, bytes]:
    aesgcm = AESGCM(key)
    nonce = os.urandom(12)
    ciphertext = aesgcm.encrypt(nonce, payload_bytes, associated_data=None)
    return nonce, ciphertext

def decrypt_payload(ciphertext: bytes, nonce: bytes, key: bytes) -> bytes:
    aesgcm = AESGCM(key)
    return aesgcm.decrypt(nonce, ciphertext, associated_data=None)

def main():
    # 1. Mandatory for CGI: Print HTTP header (the empty line separates the header from the body)
    print("Content-Type: application/json")
    print() 

    try:
        # 2. Check if the incoming request is a POST request
        if os.environ.get('REQUEST_METHOD', '') != 'POST':
            print(json.dumps({"error": "Only POST requests are allowed"}))
            return

        # 3. Read the encrypted payload (hex strings) via standard input
        content_length = int(os.environ.get('CONTENT_LENGTH', 0))
        if content_length == 0:
            print(json.dumps({"error": "No POST data received"}))
            return

        post_data = sys.stdin.read(content_length)
        client_request = json.loads(post_data)

        client_nonce = bytes.fromhex(client_request["nonce"])
        client_ciphertext = bytes.fromhex(client_request["ciphertext"])

        # 4. Decrypt: This yields the plaintext JSON (the routing payload)
        decrypted_bytes = decrypt_payload(client_ciphertext, client_nonce, SHARED_KEY)
        routing_data = json.loads(decrypted_bytes.decode('utf-8'))
        
        # 5. Extract routing variables
        target_url = routing_data.get("endpoint")
        api_token = routing_data.get("token")
        llm_payload = routing_data.get("payload")
        
        if not target_url:
            raise ValueError("Target endpoint is missing in the decrypted package.")

        # 6. Convert the LLM payload back to bytes. 
        # If the payload is empty (e.g., fetching /v1/models), setting this to None forces urllib to use a GET request.
        request_data = json.dumps(llm_payload).encode('utf-8') if llm_payload else None

        # 7. Build the request to the actual LLM API (e.g., Gemini or OpenAI)
        headers = {'Content-Type': 'application/json'}
        
        # If the client provided an authentication token, attach it
        if api_token:
            headers['Authorization'] = f'Bearer {api_token}'

        # Standard User-Agent to prevent blocks from restrictive API firewalls
        headers['User-Agent'] = 'CloakLLM-Proxy/1.0'

        req = urllib.request.Request(
            target_url, 
            data=request_data, 
            headers=headers
        )
        
        with urllib.request.urlopen(req) as response:
            llm_response_bytes = response.read()

        # 8. Encrypt the plaintext response from the LLM for the journey back to the Golang client
        resp_nonce, resp_ciphertext = encrypt_payload(llm_response_bytes, SHARED_KEY)

        # 9. Send the securely encrypted JSON package back
        secure_response = {
            "nonce": resp_nonce.hex(),
            "ciphertext": resp_ciphertext.hex()
        }
        print(json.dumps(secure_response))

    except Exception as e:
        # Include full traceback to simplify debugging server-side errors
        print(json.dumps({"error": str(e), "trace": traceback.format_exc()}))

if __name__ == "__main__":
    main()