import yaml
import requests
import os
import sys

DEFAULT_CLUSTER = "cluster"

class RESTError(Exception):
    pass

class RESTClient:
    def __init__(self, base_url, username=None, password=None, ca_cert=None, headers=None, verify=True):
        self.headers = {"Content-Type": "application/json"}
        self.base_url = base_url
        self.verify = verify
        if not base_url[-1] == "/":
            self.base_url += "/"
        self.session = requests.Session()
        if username is not None:
            self.session.auth = (username, password)
        if ca_cert is not None:
            self.session.verify = ca_cert
        if headers is not None:
            self.headers.update(headers)


    def exec(self, method, path, filename):
        if filename is not "":
            with open(filename) as data:
                response = self.session.request(method, self.base_url + path, data=os.path.expandvars(data.read()), headers=self.headers, verify=self.verify)
        else:
            response = self.session.request(method, self.base_url + path, headers=self.headers, verify=self.verify)

        if response.status_code == 200:
            print(method, path, "- 200 OK")
        elif method == "DELETE" and response.status_code == 404:
            print(method, path, "- 404 Skipping")
        elif (path.endswith("_stop") or path.endswith("_close")) and response.status_code == 404:
            print(method, path, "- 404 Skipping")
        else:
            # Check if the resource already exists
            resource_exists = False
            try:
                for cause in response.json()["error"]["root_cause"]:
                    if cause["type"] == "resource_already_exists_exception":
                        resource_exists = True
            except (KeyError, ValueError, TypeError):
                pass

            # X-Pack trial license variant of "already exists"
            try:
                if response.json()["error_message"] == "Operation failed: Trial was already activated.":
                    resource_exists = True
            except (KeyError, ValueError, TypeError):
                pass

            if resource_exists:
                print(method, path, "- Already Exists!")
            else:
                raise RESTError("%s %s - %s %s" % (method, path, response.status_code, response.text))

if __name__ == '__main__':
    if len(sys.argv) == 2 and sys.argv[1] == "--version":
        with open("./version.txt") as f:
            print("Version: ", f.read())
            sys.exit(0)
    elastic_url = "%s://%s:%s" % (os.getenv("ELASTIC_SCHEME", "https"), os.environ["ELASTIC_HOST"], os.getenv("ELASTIC_PORT", "9200"))
    kibana_url = "%s://%s:%s" % (os.getenv("KIBANA_SCHEME", "https"), os.environ["KIBANA_HOST"], os.getenv("KIBANA_PORT", "5601"))
    verify = os.getenv("INSECURE_SKIP_VERIFY") != "yes"
    user = os.getenv("USER", None)
    password = os.getenv("PASSWORD", None)
    es_ca_cert = os.getenv("ES_CA_CERT", None)
    kb_ca_cert = os.getenv("KB_CA_CERT", es_ca_cert) # Fall back on default behavior where kb and es use the same cert.
    os.putenv("CLUSTER_NAME", os.getenv("CLUSTER_NAME", DEFAULT_CLUSTER)) # set default cluster name if needed

    elastic = RESTClient(elastic_url, username=user, password=password, ca_cert=es_ca_cert, verify=verify)

    # Optionally, start the X-Pack trial (an XPack license is required for the ML jobs.)
    install_trial = os.getenv("START_XPACK_TRIAL", "false").lower()
    if install_trial in ["true", "enable", "yes", "on"]:
        elastic.exec("POST", "_xpack/license/start_trial?acknowledge=true", "")

    # Kibana requires kbn-xsrf header to mitigate cross-site request forgery
    kibana = RESTClient(kibana_url, username=user, password=password, ca_cert=kb_ca_cert,
            headers={"kbn-xsrf": "reporting"}, verify=verify)
    with open("./config.yaml") as f:
        cfg = yaml.load(f)
    try:
        for l in cfg["elasticsearch"]:
            elastic.exec(l[0], os.path.expandvars(l[1]), l[2])
        for l in cfg["kibana"]:
            kibana.exec(l[0], os.path.expandvars(l[1]), l[2])
    except RESTError as e:
        print("Failed to install")
        print(e)
        sys.exit(1)

