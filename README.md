# Voltron
Components for managing multiple clusters through a single management plane. 

There are currently two components: 
* Voltron - a backend server component running in Management Plane
* Guardian - an agent running in each App Cluster that communicates with Voltron and proxies requests to its local Kube API server

## Build and deploy

Build all components:

```
make all
```

Push images
```
make cd CONFIRM=true BRANCH_NAME=branch-name
# or, automatic current branch
make cd CONFIRM=true
```

## Guardian

### Configurations

<!-- until health check restored -->
<!--GUARDIAN_PORT | Environment | 5555 | no-->
<!--GUARDIAN_HOST | Environment | localhost | no-->
Name | Type | Default | Required
--- | --- | --- | ---
GUARDIAN_LOGLEVEL | Environment | DEBUG | no
GUARDIAN_CERT_PATH | Environment | /certs | no
GUARDIAN_VOLTRON_URL | Environment | none | yes
GUARDIAN_PROXY_TARGETS | Environment | `^/api:https://kubernetes.default`<br>`^/tigera-elasticsearch:http://localhost:8002` | yes
GUARDIAN_KEEP_ALIVE_ENABLE | Environment | true | no
GUARDIAN_KEEP_ALIVE_INTERVAL | Environment | 100 ms | no

### Build and deploy

Build guardian:

```
make guardian
```

Build image:
```
make tigera/guardian
```

Push image
```
make cd CONFIRM=true BRANCH_NAME=branch-name BUILD_IMAGES="tigera/guardian"
# or, automatic current branch
make cd CONFIRM=true BUILD_IMAGES="tigera/guardian"
```

## Voltron

### Configurations

Name | Type | Default
--- | --- | ---
VOLTRON_LOGLEVEL | Environment | DEBUG
VOLTRON_PORT | Environment | 5555
VOLTRON_HOST | Environment | any
VOLTRON_TUNNEL_PORT | Environment | 5566
VOLTRON_TUNNEL_HOST | Environment | any
VOLTRON_CERT_PATH | Environment | /certs
VOLTRON_TEMPLATE_PATH | Environment | /tmp/guardian.yaml
VOLTRON_PUBLIC_IP | Environment | 127.0.0.1:32453
VOLTRON_K8S_CONFIG_PATH | Environment | <empty string>
VOLTRON_AUTHN_ON | Environment | true
VOLTRON_KEEP_ALIVE_ENABLE | Environment | true | no
VOLTRON_KEEP_ALIVE_INTERVAL | Environment | 100 ms | no

### Build and deploy

Build voltron:

```
make voltron
```

Build image:
```
make tigera/voltron
```

Push image
```
make cd CONFIRM=true BRANCH_NAME=branch-name BUILD_IMAGES="tigera/voltron"
# or, automatic current branch
make cd CONFIRM=true BUILD_IMAGES="tigera/voltron"
```

# Deploy a demo using CRC clusters

## Deploy a management cluster


![](images/arch1.png)

1. Build the voltron manifest

```
make manifests BRANCH_NAME=master -B
```

2. Build a management cluster with terraform and [calico-ready-clusters](https://github.com/tigera/calico-ready-clusters/tree/master/kubeadm/1.6)

```
terraform workspace new <clustername>
terraform apply -var prefix=<username-clustername>
cp master_ssh_key <clustername>-master_ssh
cp admin.conf <clustername>-admin.conf
```

3. Copy voltron manifest, network policy, licence and docker credentials

- [voltron.yaml](manifests)
- [tigera-engineering-test-auth.json](https://tigera.atlassian.net/wiki/spaces/ENG/pages/44925032/Files+we+should+neither+lose+nor+give+away+test+licenses+secrets+etc)
- [new-test-customer-license.yaml](https://tigera.atlassian.net/wiki/spaces/ENG/pages/44925032/Files+we+should+neither+lose+nor+give+away+test+licenses+secrets+etc)
- [install-cnx.sh](scripts/demo/install-cnx.sh)
- [allow-cnx.allow-voltron-access.yaml](manifests/allow-cnx.allow-voltron-access.yaml)

4. Run the install-cnx script with the following parameters

```
AUTH_TYPE=OIDC OIDC_CLIENT_ID=114913197347-dvlgarrcfbp5dsjb2c3dkl607434eg42.apps.googleusercontent.com ./install-cnx.sh -l new-test-customer-license.yaml -c tigera-engineering-test-auth.json
```

5. Adding application clusters

```
kubectl exec -it <voltron_pod> -ncalico-monitoring -- sh /scripts/register.bash
kubectl cp calico-monitoring/<voltorn_pod>:/guardian1.yaml guardian1.yaml
kubectl cp calico-monitoring/<voltorn_pod>:/guardian2.yaml guardian2.yaml
```

# Project structure

## 1. Go Directories

### `/cmd`

Main applications for this project.

The directory name for each application should match the name of the executable you want to have (e.g., `/cmd/myapp`).

Don't put a lot of code in the application directory. If you think the code can be imported and used in other projects, then it should live in the `/pkg` directory. If the code is not reusable or if you don't want others to reuse it, put that code in the `/internal` directory. You'll be surprised what others will do, so be explicit about your intentions!

It's common to have a small `main` function that imports and invokes the code from the `/internal` and `/pkg` directories and nothing else.

See the [`/cmd`](cmd/README.md) directory for examples.

### `/internal`

Private application and library code. This is the code you don't want others importing in their applications or libraries.

Put your actual application code in the `/internal/app` directory (e.g., `/internal/app/myapp`) and the code shared by those apps in the `/internal/pkg` directory (e.g., `/internal/pkg/myprivlib`).

### `/pkg`

Library code that's ok to use by external applications (e.g., `/pkg/mypubliclib`). Other projects will import these libraries expecting them to work, so think twice before you put something here :-)

It's also a way to group Go code in one place when your root directory contains lots of non-Go components and directories making it easier to run various Go tools (as mentioned in the [`Best Practices for Industrial Programming`](https://www.youtube.com/watch?v=PTE4VJIdHPg) from GopherCon EU 2018).

See the [`/pkg`](pkg/README.md) directory if you want to see which popular Go repos use this project layout pattern. This is a common layout pattern, but it's not universally accepted and some in the Go community don't recommend it. 

### `/vendor`

Application dependencies (managed your favorite dependency management tool like [`dep`](https://github.com/golang/dep)).

Don't commit your application dependencies if you are building a library.

## 2. Common Application Directories

### `/scripts`

Scripts to perform various build, install, analysis, etc operations.

These scripts keep the root level Makefile small and simple (e.g., `https://github.com/hashicorp/terraform/blob/master/Makefile`).

See the [`/scripts`](scripts/README.md) directory for examples.

### `/test`

Additional external test apps and test data. Feel free to structure the `/test` directory anyway you want. For bigger projects it makes sense to have a data subdirectory. For example, you can have `/test/data` or `/test/testdata` if you need Go to ignore what's in that directory. Note that Go will also ignore directories or files that begin with "." or "_", so you have more flexibility in terms of how you name your test data directory.

See the [`/test`](test/README.md) directory for examples.

## 3. Other Directories

### `/docs`

Design and user documents (in addition to your godoc generated documentation).

See the [`/docs`](docs/README.md) directory for examples.

## 4. Root-level files

### `Dockerfile`

### `Makefile`

## 5. Go Modules 
This template assumes you are using Go modules. Read the following sections in the Go modules documentation to learn about how to use it: 

* Go module [concepts](https://github.com/golang/go/wiki/Modules#new-concepts) 
* How to use [Go modules](https://github.com/golang/go/wiki/Modules#how-to-use-modules)
