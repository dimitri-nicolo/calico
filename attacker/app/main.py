from flask import Flask, request, redirect, url_for
import subprocess
import base64

app = Flask(__name__)

@app.route('/')
def main():
    out = '<html><head><title>Compromised Website</title></head><body><h1>Compromised Website</h1></br>\
           <h3>Start Here</h3>'
    return out

if __name__ == '__main__':
    app.run(debug=True, host='0.0.0.0', port=80)


