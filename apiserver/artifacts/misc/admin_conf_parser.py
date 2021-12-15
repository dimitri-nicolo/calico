#!/usr/bin/env python

import yaml
import base64

with open("/etc/kubernetes/admin.conf", 'r') as stream:
    try:
        admin = yaml.load(stream)
        user = admin['users'][0]
        client_cert = base64.b64decode(user['user']['client-certificate-data'])
        client_key = base64.b64decode(user['user']['client-key-data'])
        f = open("/var/tmp/client.includesprivatekey.pem","w+")
        f.write(client_cert)
        f.write(client_key)
        f.close() 
    except yaml.YAMLError as exc:
        print(exc)
