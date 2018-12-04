import yaml
import requests
import os
import sys

class RESTError(Exception):
    pass

class RESTClient:
    headers = {"Content-Type": "application/json"}

    def __init__(self, base_url, username=None, password=None, ca_cert=None):
        self.base_url = base_url
        if not base_url[-1] == "/":
            self.base_url += "/"
        self.session = requests.Session() 
        if username is not None:
            self.session.auth = (username, password)
        if ca_cert is not None:
            self.session.verify = ca_cert

    def exec(self, method, path, filename):
        with open(filename) as data:
            response = self.session.request(method, self.base_url + path, data=data, headers=self.headers)
            if response.status_code == 200:
                print(method, path, "- 200 OK")
            else:
                raise RESTError("%s %s - %s %s" % (method, path, response.status_code, response.text))

if __name__ == '__main__':

    elastic_url = "https://%s:%s" % (os.environ["ELASTIC_HOST"], os.getenv("ELASTIC_PORT", "9200"))
    user = os.getenv("USER", None)
    password = os.getenv("PASSWORD", None)
    ca_cert = os.getenv("CA_CERT", None)

    elastic = RESTClient(elastic_url, user, password, ca_cert)
    with open("./config.yaml") as f:
        cfg = yaml.load(f)
    try:
        for l in cfg["elasticsearch"]:
            elastic.exec(l[0], l[1], l[2])
    except RESTError as e:
        print("Failed to install")
        print(e)
        sys.exit(1)
