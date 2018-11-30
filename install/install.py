import yaml
import requests
from docopt import docopt

doc = """Elastic Job Installer

Usage:
  installer.py <elastic_url> <kibana_url> [options]

Options:
  <elastic_url>             Elasticsearch base URL, e.g. https://elasticsearch.mydomain:9200
  <kibana_url>              Kibana base URL, e.g. https://kibana.mydomain:5601
  -u --user=<user>          User name for authentication with Elastic / Kibana
  -p --password=<password>  Password for authentication with Elastic / Kibana
  -c --ca-cert=<ca-cert>    Path to certificate authority root certificate. Use if Elastic / Kibana are HTTPS, but don't use a public root of trust.
  -f --config=<config>      Path to the config file [default: ./config.yaml].
  -h --help                 Print this screen.
"""

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
                print(method, path, "-", response.status_code, response.text)

if __name__ == '__main__':
    arguments = docopt(doc, help=True)
    elastic = RESTClient(arguments["<elastic_url>"], arguments["--user"], arguments["--password"], arguments["--ca-cert"])
    with open(arguments["--config"]) as f:
        cfg = yaml.load(f)
    for l in cfg["elasticsearch"]:
        elastic.exec(l[0], l[1], l[2])
