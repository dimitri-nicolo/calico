version: v1.0
name: Calico

execution_time_limit:
  hours: 4

agent:
  machine:
    type: e1-standard-2
    os_image: ubuntu2004

auto_cancel:
  running:
    when: "branch != 'master'"
  queued:
    when: "branch != 'master'"

promotions:
# Cleanup after ourselves if we are stopped-short.
- name: Cleanup
  pipeline_file: cleanup.yml
  auto_promote:
    when: "result = 'stopped'"
# Have separate promotions for publishing images so we can re-run
# them individually if they fail, and so we can run them in parallel.
- name: Push apiserver images
  pipeline_file: push-images/apiserver.yml
  auto_promote:
    when: "branch =~ 'master|release-.*'"
- name: Push cni-plugin images
  pipeline_file: push-images/cni-plugin.yml
  auto_promote:
    when: "branch =~ 'master|release-'"
- name: Push kube-controllers images
  pipeline_file: push-images/kube-controllers.yml
  auto_promote:
    when: "branch =~ 'master|release-'"
- name: Push calicoctl images
  pipeline_file: push-images/calicoctl.yml
  auto_promote:
    when: "branch =~ 'master|release-'"
- name: Push calicoq images
  pipeline_file: push-images/calicoq.yml
  auto_promote:
    when: "branch =~ 'master|release-'"
- name: Push typha images
  pipeline_file: push-images/typha.yml
  auto_promote:
    when: "branch =~ 'master|release-'"
- name: Push ALP images
  pipeline_file: push-images/alp.yml
  auto_promote:
    when: "branch =~ 'master|release-'"
- name: Push calico/node images
  pipeline_file: push-images/node.yml
  auto_promote:
    when: "branch =~ 'master|release-'"
- name: Push anomaly-detection-api images
  pipeline_file: push-images/anomaly-detection-api.yml
  auto_promote:
    when: "branch =~ 'master|release-'"
- name: Push compliance images
  pipeline_file: push-images/compliance.yml
  auto_promote:
    when: "branch =~ 'master|release-'"
- name: Push continuous-policy-recommendation images
  pipeline_file: push-images/continuous-policy-recommendation.yml
  auto_promote:
    when: "branch =~ 'master|release-'"
- name: Push deep-packet-inspection images
  pipeline_file: push-images/deep-packet-inspection.yml
  auto_promote:
    when: "branch =~ 'master|release-'"
- name: Push egress-gateway images
  pipeline_file: push-images/egress-gateway.yml
  auto_promote:
    when: "branch =~ 'master|release-'"
- name: Push es-gateway images
  pipeline_file: push-images/es-gateway.yml
  auto_promote:
    when: "branch =~ 'master|release-'"
- name: Push es-proxy images
  pipeline_file: push-images/es-proxy.yml
  auto_promote:
    when: "branch =~ 'master|release-'"
- name: Push firewall-integration images
  pipeline_file: push-images/firewall-integration.yml
  auto_promote:
    when: "branch =~ 'master|release-'"
- name: Push honeypod-controller images
  pipeline_file: push-images/honeypod-controller.yml
  auto_promote:
    when: "branch =~ 'master|release-'"
- name: Push intrusion-detection-controller images
  pipeline_file: push-images/intrusion-detection-controller.yml
  auto_promote:
    when: "branch =~ 'master|release-'"
- name: Push l7-collector images
  pipeline_file: push-images/l7-collector.yml
  auto_promote:
    when: "branch =~ 'master|release-'"
- name: Push license-agent images
  pipeline_file: push-images/license-agent.yml
  auto_promote:
    when: "branch =~ 'master|release-'"
- name: Push packetcapture images
  pipeline_file: push-images/packetcapture.yml
  auto_promote:
    when: "branch =~ 'master|release-'"
- name: Push prometheus-service images
  pipeline_file: push-images/prometheus-service.yml
  auto_promote:
    when: "branch =~ 'master|release-'"
- name: Push ts-queryserver images
  pipeline_file: push-images/ts-queryserver.yml
  auto_promote:
    when: "branch =~ 'master|release-'"
- name: Push voltron images
  pipeline_file: push-images/voltron.yml
  auto_promote:
    when: "branch =~ 'master|release-'"
- name: Publish Helm Charts
  pipeline_file: docs/helm-charts.yml
  auto_promote:
    when: "branch =~ 'master|release-'"
- name: Publish docs
  pipeline_file: docs/deploy-site.yml
  auto_promote:
    when: "branch =~ 'master|release-'"

global_job_config:
  secrets:
  - name: docker-hub
  - name: banzai-secrets
  - name: private-repo
  prologue:
    commands:
    # Correct permissions since they are too open by default:
    - chmod 0600 ~/.keys/*
    # Add the key to the ssh agent:
    - ssh-add ~/.keys/*
    - checkout
    - export REPO_DIR="$(pwd)"
    - mkdir artifacts
    # Semaphore is doing shallow clone on a commit without tags.
    # unshallow it for GIT_VERSION:=$(shell git describe --tags --dirty --always)
    - git fetch --unshallow
    # Semaphore mounts a copy-on-write FS as /var/lib/docker in order to provide a pre-loaded cache of
    # some images. However, the cache is not useful to us and the copy-on-write FS is a big problem given
    # how much we churn docker containers during the build.  Disable it.
    - sudo systemctl stop docker
    - sudo umount /var/lib/docker && sudo killall qemu-nbd || true
    - sudo systemctl start docker
    # Free up space on the build machine.
    - sudo rm -rf ~/{.kerl,.kiex,.npm,.nvm,.phpbrew,.rbenv,.sbt} /opt/{apache-maven*,firefox*,scala} /usr/lib/jvm /usr/local/{aws2,golang,phantomjs*}
    - echo $DOCKERHUB_PASSWORD | docker login --username "$DOCKERHUB_USERNAME" --password-stdin
    # Disable initramfs update to save space on the Semaphore VM (and we don't need it because we're not going to reboot).
    - sudo apt-get install -y -u crudini
    - sudo crudini --set /etc/initramfs-tools/update-initramfs.conf '' update_initramfs no
    - cat /etc/initramfs-tools/update-initramfs.conf
  epilogue:
    commands:
    - cd "$REPO_DIR"
    - .semaphore/publish-artifacts

blocks:

- name: "Prerequisites"
  dependencies: []
  task:
    jobs:
    - name: "Check generated files"
      commands:
      - make generate
      - make check-dirty

- name: "API"
  run:
    when: "${FORCE_RUN} or change_in(['/*', '/api/'], {exclude: ['/**/.gitignore', '/**/README.md', '/**/LICENSE']})"
  execution_time_limit:
    minutes: 30
  dependencies: ["Prerequisites"]
  task:
    prologue:
      commands:
      - cd api
    jobs:
    - name: "make ci"
      commands:
      - ../.semaphore/run-and-monitor make-ci.log make ci

- name: "apiserver"
  run:
    when: "${FORCE_RUN} or change_in(['/*', '/libcalico-go/', '/api/', '/apiserver/', '/licensing/', '/hack/test/certs/'], {exclude: ['/**/.gitignore', '/**/README.md', '/**/LICENSE']})"
  execution_time_limit:
    minutes: 30
  dependencies: ["Prerequisites"]
  task:
    agent:
      machine:
        type: e1-standard-4
        os_image: ubuntu2004
    prologue:
      commands:
      - cd apiserver
    jobs:
    - name: "make ci"
      commands:
      - ../.semaphore/run-and-monitor make-ci.log make ci

- name: "apiserver: build all architectures"
  run:
    when: "${FORCE_RUN} or change_in(['/*', '/api/', '/apiserver/'], {exclude: ['/**/.gitignore', '/**/README.md', '/**/LICENSE']})"
  dependencies: ["Prerequisites"]
  task:
    agent:
      machine:
        type: e1-standard-4
        os_image: ubuntu2004
    prologue:
      commands:
      - cd apiserver
    jobs:
    - name: "Build image"
      matrix:
      - env_var: ARCH
        values: [ "arm64", "ppc64le" ]
      commands:
      - ../.semaphore/run-and-monitor image-$ARCH.log make image ARCH=$ARCH

- name: "libcalico-go"
  run:
    when: "${FORCE_RUN} or change_in(['/*', '/api/', '/libcalico-go/'], {exclude: ['/**/.gitignore', '/**/README.md', '/**/LICENSE']})"
  dependencies: ["Prerequisites"]
  task:
    jobs:
    - name: "libcalico-go: tests"
      commands:
      - cd libcalico-go
      - ../.semaphore/run-and-monitor make-ci.log make ci

- name: "Typha"
  run:
    when: "${FORCE_RUN} or change_in(['/*', '/api/', '/libcalico-go/', '/typha/', '/hack/test/certs/'], {exclude: ['/**/.gitignore', '/**/README.md', '/**/LICENSE']})"
  dependencies: ["Prerequisites"]
  task:
    agent:
      machine:
        type: e1-standard-4
        os_image: ubuntu2004
    jobs:
    - name: "Typha: UT and FV tests"
      commands:
      - cd typha
      - ../.semaphore/run-and-monitor make-ci.log make ci EXCEPT=k8sfv-test
    epilogue:
      always:
        commands:
        - |
          for f in /home/semaphore/calico/typha/report/*; do
            NAME=$(basename $f)
            test-results compile --name typha-$NAME $f $NAME.json || true
          done
          for f in /home/semaphore/calico/typha/pkg/report/*; do
            NAME=$(basename $f)
            test-results compile --name typha-$NAME $f $NAME.json || true
          done
          test-results combine *.xml.json report.json || true
          artifact push job report.json -d test-results/junit.json || true
          artifact push workflow report.json -d test-results/${SEMAPHORE_PIPELINE_ID}/${SEMAPHORE_JOB_ID}.json || true
        - test-results publish /home/semaphore/calico/felix/report/k8sfv_suite.xml --name "typha-k8sfv" || true

- name: "Felix: Build"
  run:
    when: "${FORCE_RUN} or change_in(['/*', '/api/', '/libcalico-go/', '/typha/', '/felix/', '/hack/test/certs/'], {exclude: ['/**/.gitignore', '/**/README.md', '/**/LICENSE']})"
  dependencies: ["Prerequisites"]
  task:
    agent:
      machine:
        type: e1-standard-4
        os_image: ubuntu2004
    prologue:
      commands:
      - cd felix
      - cache restore go-pkg-cache
      - cache restore go-mod-cache
    jobs:
    - name: Build and run UT, k8sfv
      execution_time_limit:
        minutes: 60
      commands:
      - make build image fv-prereqs
      - 'cache store bin-${SEMAPHORE_GIT_SHA} bin'
      - 'cache store fv.test-${SEMAPHORE_GIT_SHA} fv/fv.test'
      - cache store go-pkg-cache .go-pkg-cache
      - 'cache store go-mod-cache ${HOME}/go/pkg/mod/cache'
      - docker save -o /tmp/calico-felix.tar tigera/felix:latest-amd64
      - 'cache store felix-image-${SEMAPHORE_GIT_SHA} /tmp/calico-felix.tar'
      - docker save -o /tmp/felixtest-typha.tar felix-test/typha:latest-amd64
      - 'cache store felixtest-typha-image-${SEMAPHORE_GIT_SHA} /tmp/felixtest-typha.tar'
      - docker save -o /tmp/felixtest-dns-server.tar tigera-test/dns-server:latest
      - 'cache store felixtest-dns-server-image-${SEMAPHORE_GIT_SHA} /tmp/felixtest-dns-server.tar'
      - docker save -o /tmp/felixtest-scapy.tar tigera-test/scapy:latest
      - 'cache store felixtest-scapy-image-${SEMAPHORE_GIT_SHA} /tmp/felixtest-scapy.tar'
      - ../.semaphore/run-and-monitor ut.log make ut
      - ../.semaphore/run-and-monitor k8sfv-typha.log make k8sfv-test JUST_A_MINUTE=true USE_TYPHA=true
      - ../.semaphore/run-and-monitor k8sfv-no-typha.log make k8sfv-test JUST_A_MINUTE=true USE_TYPHA=false
    - name: Static checks
      execution_time_limit:
        minutes: 60
      commands:
      - ../.semaphore/run-and-monitor static-checks.log make static-checks

- name: "Felix: Build Windows binaries"
  run:
    when: "${FORCE_RUN} or change_in(['/*', '/api/', '/libcalico-go/', '/typha/', '/felix/'], {exclude: ['/**/.gitignore', '/**/README.md', '/**/LICENSE']})"
  dependencies: ["Prerequisites"]
  task:
    jobs:
    - name: "build Windows binaries"
      commands:
      - cd felix
      - make bin/calico-felix.exe fv/win-fv.exe

- name: "Felix: Windows FV"
  run:
    when: "${FORCE_RUN} or change_in(['/*', '/api/', '/libcalico-go/', '/typha/', '/felix/', '/hack/test/certs/'], {exclude: ['/**/.gitignore', '/**/README.md', '/**/LICENSE']})"
  dependencies: ["Felix: Build Windows binaries"]
  task:
    secrets:
    - name: banzai-secrets
    - name: private-repo
    prologue:
      commands:
      # Load the github access secrets.  First fix the permissions.
      - chmod 0600 ~/.keys/*
      - ssh-add ~/.keys/*
      # Prepare aws configuration.
      - pip install --upgrade --user awscli
      - export REPORT_DIR=~/report
      - export LOGS_DIR=~/fv.log
      - export SHORT_WORKFLOW_ID=$(echo ${SEMAPHORE_WORKFLOW_ID} | sha256sum | cut -c -8)
      - export CLUSTER_NAME=sem-${SEMAPHORE_PROJECT_NAME}-pr${SEMAPHORE_GIT_PR_NUMBER}-${WINDOWS_VERSION}-${BACKEND}-${SHORT_WORKFLOW_ID}
      - export KEYPAIR_NAME=${CLUSTER_NAME}
      - echo CLUSTER_NAME=${CLUSTER_NAME}
      - sudo apt-get install -y putty-tools
      - git clone git@github.com:tigera/process.git ~/process
      - cd felix
      - make bin/calico-felix.exe fv/win-fv.exe
    epilogue:
      always:
        commands:
        - artifact push job ${REPORT_DIR} --destination semaphore/test-results --expire-in ${SEMAPHORE_ARTIFACT_EXPIRY} || true
        - artifact push job ${LOGS_DIR} --destination semaphore/logs --expire-in ${SEMAPHORE_ARTIFACT_EXPIRY} || true
        - aws ec2 delete-key-pair --key-name ${KEYPAIR_NAME} || true
        - cd ~/process/testing/winfv && NAME_PREFIX="${CLUSTER_NAME}" ./setup-fv.sh -q -u
    env_vars:
    - name: SEMAPHORE_ARTIFACT_EXPIRY
      value: 2w
    - name: MASTER_CONNECT_KEY_PUB
      value: master_ssh_key.pub
    - name: MASTER_CONNECT_KEY
      value: master_ssh_key
    - name: WIN_PPK_KEY
      value: win_ppk_key
    - name: K8S_VERSION
      value: 1.22.1
    - name: WINDOWS_VERSION
      value: "1809"
    jobs:
    - name: Windows FV
      commands:
      - ./.semaphore/run-win-fv
      matrix:
      - env_var: WINDOWS_VERSION
        values: ["20H2"]
      - env_var: BACKEND
        values: ["bgp", "vxlan"]

- name: "Felix: FV Tests"
  run:
    when: "${FORCE_RUN} or change_in(['/*', '/api/', '/libcalico-go/', '/typha/', '/felix/'], {exclude: ['/**/.gitignore', '/**/README.md', '/**/LICENSE']})"
  dependencies: ["Felix: Build"]
  task:
    agent:
      machine:
        type: e1-standard-4
        os_image: ubuntu2004
    prologue:
      commands:
      - cd felix
      - cache restore go-pkg-cache
      - cache restore go-mod-cache
      - 'cache restore bin-${SEMAPHORE_GIT_SHA}'
      - 'cache restore fv.test-${SEMAPHORE_GIT_SHA}'
      - 'cache restore felix-image-${SEMAPHORE_GIT_SHA}'
      - 'cache restore felixtest-typha-image-${SEMAPHORE_GIT_SHA}'
      - 'cache restore felixtest-dns-server-image-${SEMAPHORE_GIT_SHA}'
      - 'cache restore scapy-image-${SEMAPHORE_GIT_SHA}'
      - |-
        if [ -s /etc/docker/daemon.json  ]; then
        sudo sed -i '$d' /etc/docker/daemon.json && sudo sed -i '$s/$/,/' /etc/docker/daemon.json && sudo bash -c ' cat >> /etc/docker/daemon.json << EOF
          "ipv6": true,
          "fixed-cidr-v6": "2001:db8:1::/64"
        }
        EOF
        ' ; else sudo bash -c ' cat > /etc/docker/daemon.json << EOF
        {
          "ipv6": true,
          "fixed-cidr-v6": "2001:db8:1::/64"
        }
        EOF
        ' ; fi
      - sudo systemctl restart docker
      # Load in the docker images pre-built by the build job.
      - docker load -i /tmp/calico-felix.tar
      - docker tag tigera/felix:latest-amd64 felix:latest-amd64
      - rm /tmp/calico-felix.tar
      - docker load -i /tmp/felixtest-typha.tar
      - docker tag felix-test/typha:latest-amd64 typha:latest-amd64
      - rm /tmp/felixtest-typha.tar
      - docker load -i /tmp/felixtest-dns-server.tar
      - rm /tmp/felixtest-dns-server.tar
      - docker load -i /tmp/felixtest-scapy.tar
      - rm /tmp/felixtest-scapy.tar
      # Pre-loading the IPIP module prevents a flake where the first felix to use IPIP loads the module and
      # routing in that first felix container chooses different source IPs than the tests are expecting.
      - sudo modprobe ipip
    jobs:
    - name: FV Test matrix
      execution_time_limit:
        minutes: 120
      commands:
      - make check-wireguard
      - ../.semaphore/run-and-monitor fv-${SEMAPHORE_JOB_INDEX}.log make fv-no-prereqs FV_BATCHES_TO_RUN="${SEMAPHORE_JOB_INDEX}" FV_NUM_BATCHES=${SEMAPHORE_JOB_COUNT}
      parallelism: 3
    epilogue:
      always:
        commands:
        - ./.semaphore/collect-artifacts
        - ./.semaphore/publish-artifacts
        - test-results publish /home/semaphore/calico/felix/report/fv_suite.xml --name "felix-fv-${SEMAPHORE_JOB_INDEX}" || true

- name: "Felix: BPF UT/FV tests on new kernel"
  run:
    when: "${FORCE_RUN} or change_in(['/*', '/api/', '/libcalico-go/', '/typha/', '/felix/'], {exclude: ['/**/.gitignore', '/**/README.md', '/**/LICENSE']})"
  dependencies: ["Prerequisites"]
  task:
    prologue:
      commands:
      - cd felix
      - export GOOGLE_APPLICATION_CREDENTIALS=$HOME/secrets/secret.google-service-account-key.json
      - export SHORT_WORKFLOW_ID=$(echo ${SEMAPHORE_WORKFLOW_ID} | sha256sum | cut -c -8)
      - export ZONE=europe-west3-c
      - export VM_PREFIX=sem-${SEMAPHORE_PROJECT_NAME}-${SHORT_WORKFLOW_ID}-
      - echo VM_PREFIX=${VM_PREFIX}
      - export REPO_NAME=$(basename $(pwd))
      - export NUM_FV_BATCHES=8
      - mkdir artifacts
      - ./.semaphore/create-test-vms ${VM_PREFIX}
    jobs:
    - name: UT/FV tests on new kernel
      execution_time_limit:
        minutes: 120
      commands:
      - ./.semaphore/run-tests-on-vms ${VM_PREFIX}
    epilogue:
      always:
        commands:
        - ./.semaphore/collect-artifacts-from-vms ${VM_PREFIX}
        - ./.semaphore/publish-artifacts
        - ./.semaphore/clean-up-vms ${VM_PREFIX}
    secrets:
    - name: google-service-account-for-gce

- name: "confd: tests"
  run:
    when: "${FORCE_RUN} or change_in(['/*', '/api/', '/libcalico-go/', '/confd/', '/hack/test/certs/'], {exclude: ['/**/.gitignore', '/**/README.md', '/**/LICENSE']})"
  dependencies: ["Prerequisites"]
  task:
    prologue:
      commands:
      - cd confd
    jobs:
    - name: "confd: CI"
      execution_time_limit:
        minutes: 60
      commands:
        - ../.semaphore/run-and-monitor ci.log make ci

- name: "Node: Tests"
  run:
    when: "${FORCE_RUN} or change_in(['/*', '/api/', '/libcalico-go/', '/typha/', '/felix/', '/confd/', '/bird/', '/pod2daemon/', '/node/', '/licensing/', '/hack/test/certs/'], {exclude: ['/**/.gitignore', '/**/README.md', '/**/LICENSE']})"
  dependencies: ["Prerequisites"]
  task:
    agent:
      machine:
        type: e1-standard-8
        os_image: ubuntu2004
    secrets:
    # Mount a secret for pulling images from GCR, and a license for the k8s FVs
    - name: tigera-dev-ci-pull-credentials
    - name: google-service-account-for-gcr
    - name: test-customer-license
    prologue:
      commands:
      - cd node
    jobs:
    - name: "Node: CI"
      commands:
      - ../.semaphore/run-and-monitor ci.log make ci
    - name: "Node: k8s-test"
      commands:
      - ../.semaphore/run-and-monitor k8s-test.log make k8s-test
    - name: "Node: dual-tor"
      commands:
      - ../.semaphore/run-and-monitor dual-tor-test.log make dual-tor-test
    epilogue:
      always:
        commands:
        - test-results publish ./report/nosetests.xml --name "node-ci" || true
        - test-results publish ./report/k8s-tests.xml --name "node-k8s-test" || true

- name: "Node/egress gateway tests"
  run:
    when: "${FORCE_RUN} or change_in(['/*', '/api/', '/libcalico-go/', '/typha/', '/felix/', '/confd/', '/bird/', '/pod2daemon/', '/node/', '/egress-gateway/', '/licensing/', '/hack/test/certs/'], {exclude: ['/**/.gitignore', '/**/README.md', '/**/LICENSE']})"
  dependencies: ["Prerequisites"]
  task:
    agent:
      machine:
        type: e1-standard-8
        os_image: ubuntu2004
    secrets:
    # Mount a secret for pulling images from GCR, and a license for the k8s FVs
    - name: tigera-dev-ci-pull-credentials
    - name: google-service-account-for-gcr
    - name: test-customer-license
    prologue:
      commands:
      - cd node
    jobs:
    - name: "Node: egress-ip no-overlay"
      commands:
      - ../.semaphore/run-and-monitor egress-ip-test.log make egress-ip-test K8ST_TO_RUN="-A egress_ip_no_overlay"
    - name: "Node: egress-ip ipip"
      commands:
      - ../.semaphore/run-and-monitor egress-ip-test.log make egress-ip-test K8ST_TO_RUN="-A egress_ip_ipip"
    - name: "Node: egress-ip vxlan"
      commands:
      - ../.semaphore/run-and-monitor egress-ip-test.log make egress-ip-test K8ST_TO_RUN="-A egress_ip_vxlan"
    epilogue:
      always:
        commands:
        - test-results publish ./report/nosetests.xml --name "node-ci" || true
        - test-results publish ./report/k8s-tests.xml --name "node-k8s-test" || true

- name: "Node: build all architectures"
  run:
    when: "change_in(['/felix/', '/confd/', '/node/'])"
  dependencies: ["Prerequisites"]
  task:
    agent:
      machine:
        type: e1-standard-4
        os_image: ubuntu2004
    prologue:
      commands:
      - cd node
    jobs:
# Non-AMD64 images not supported in Enterprise.
#
#    - name: "Build image"
#      matrix:
#      # TODO: s390x builds of calico/node are not working.
#      - env_var: ARCH
#        values: [ "arm64", "armv7", "ppc64le" ]
#      commands:
#      - ../.semaphore/run-and-monitor image-$ARCH.log make image ARCH=$ARCH
    - name: "Build Windows archive"
      commands:
      - ../.semaphore/run-and-monitor build-windows-archive.log make build-windows-archive

- name: "e2e tests"
  run:
    when: "${FORCE_RUN} or change_in(['/*', '/api/', '/libcalico-go/', '/typha/', '/felix/', '/confd/', '/bird/', '/pod2daemon/', '/node/', '/licensing/'], {exclude: ['/**/.gitignore', '/**/README.md', '/**/LICENSE']})"
  dependencies: ["Prerequisites"]
  task:
    secrets:
    # Since we create a local kind cluster, we need the test license.
    - name: test-customer-license
    agent:
      machine:
        type: e1-standard-8
        os_image: ubuntu2004
    jobs:
    - name: "sig-network conformance"
      env_vars:
      - name: E2E_FOCUS
        value: "sig-network.*Conformance"
      commands:
      - .semaphore/run-and-monitor e2e-test.log make e2e-test

- name: "kube-controllers: Tests"
  run:
    when: "${FORCE_RUN} or change_in(['/*', '/api/', '/libcalico-go/', '/kube-controllers/', '/licensing/', '/hack/test/certs/'], {exclude: ['/**/.gitignore', '/**/README.md', '/**/LICENSE']})"
  dependencies: ["Prerequisites"]
  task:
    prologue:
      commands:
      - cd kube-controllers
    jobs:
    - name: "kube-controllers: tests"
      commands:
      - ../.semaphore/run-and-monitor ci.log make ci

- name: "pod2daemon"
  run:
    when: "${FORCE_RUN} or change_in(['/*', '/pod2daemon/'], {exclude: ['/**/.gitignore', '/**/README.md', '/**/LICENSE']})"
  dependencies: ["Prerequisites"]
  task:
    prologue:
      commands:
      - cd pod2daemon
    jobs:
    - name: "pod2daemon tests"
      commands:
      - ../.semaphore/run-and-monitor ci.log make ci

- name: "app-policy"
  run:
    when: "${FORCE_RUN} or change_in(['/*', '/app-policy/', '/felix/', '/licensing/'], {exclude: ['/**/.gitignore', '/**/README.md', '/**/LICENSE']})"
  dependencies: ["Prerequisites"]
  task:
    prologue:
      commands:
      - cd app-policy
    jobs:
    - name: "app-policy tests"
      commands:
      - ../.semaphore/run-and-monitor ci.log make ci

- name: "calicoctl"
  run:
    when: "${FORCE_RUN} or change_in(['/*', '/calicoctl/', '/libcalico-go/', '/api/', '/licensing/', '/hack/test/certs/'], {exclude: ['/**/.gitignore', '/**/README.md', '/**/LICENSE']})"
  dependencies: ["Prerequisites"]
  task:
    prologue:
      commands:
      - cd calicoctl
    jobs:
    - name: "calicoctl tests"
      commands:
      - ../.semaphore/run-and-monitor ci.log make ci

- name: "cni-plugin: Windows"
  run:
    when: "${FORCE_RUN} or change_in(['/*', '/cni-plugin/', '/libcalico-go/'], {exclude: ['/**/.gitignore', '/**/README.md', '/**/LICENSE']})"
  dependencies: ["Prerequisites"]
  task:
    secrets:
    - name: banzai-secrets
    - name: private-repo
    prologue:
      commands:
      # Load the github access secrets.  First fix the permissions.
      - chmod 0600 ~/.keys/*
      - ssh-add ~/.keys/*
      # Prepare aws configuration.
      - pip install --upgrade --user awscli
      - export REPORT_DIR=~/report
      - export LOGS_DIR=~/fv.log
      - export SHORT_WORKFLOW_ID=$(echo ${SEMAPHORE_WORKFLOW_ID} | sha256sum | cut -c -8)
      - export CLUSTER_NAME=sem-${SEMAPHORE_PROJECT_NAME}-pr${SEMAPHORE_GIT_PR_NUMBER}-${CONTAINER_RUNTIME}-${SHORT_WORKFLOW_ID}
      - export KEYPAIR_NAME=${CLUSTER_NAME}
      - echo CLUSTER_NAME=${CLUSTER_NAME}
      - sudo apt-get install -y putty-tools
      - git clone git@github.com:tigera/process.git ~/process
      - cd cni-plugin
      - ../.semaphore/run-and-monitor build.log make bin/windows/calico.exe bin/windows/calico-ipam.exe bin/windows/win-fv.exe
    epilogue:
      always:
        commands:
        - artifact push job ${REPORT_DIR} --destination semaphore/test-results --expire-in ${SEMAPHORE_ARTIFACT_EXPIRY} || true
        - artifact push job ${LOGS_DIR} --destination semaphore/logs --expire-in ${SEMAPHORE_ARTIFACT_EXPIRY} || true
        - aws ec2 delete-key-pair --key-name ${KEYPAIR_NAME} || true
        - cd ~/process/testing/winfv && NAME_PREFIX="${CLUSTER_NAME}" ./setup-fv.sh -q -u
    env_vars:
    - name: SEMAPHORE_ARTIFACT_EXPIRY
      value: 2w
    - name: AWS_DEFAULT_REGION
      value: us-west-2
    - name: MASTER_CONNECT_KEY_PUB
      value: master_ssh_key.pub
    - name: MASTER_CONNECT_KEY
      value: master_ssh_key
    - name: WIN_PPK_KEY
      value: win_ppk_key
    - name: K8S_VERSION
      value: 1.22.6
    jobs:
    - name: Docker - Windows FV
      execution_time_limit:
        minutes: 60
      commands:
      - ../.semaphore/run-and-monitor win-fv-docker.log ./.semaphore/run-win-fv.sh
      env_vars:
      - name: CONTAINER_RUNTIME
        value: docker
    - name: Containerd - Windows FV
      execution_time_limit:
        minutes: 60
      commands:
      - ../.semaphore/run-and-monitor win-fv-containerd.log ./.semaphore/run-win-fv.sh
      env_vars:
      - name: CONTAINER_RUNTIME
        value: containerd
      - name: CONTAINERD_VERSION
        value: 1.6.0

- name: "cni-plugin"
  run:
    when: "${FORCE_RUN} or change_in(['/*', '/cni-plugin/', '/libcalico-go/', '/hack/test/certs/'], {exclude: ['/**/.gitignore', '/**/README.md', '/**/LICENSE']})"
  dependencies: ["Prerequisites"]
  task:
    prologue:
      commands:
      - cd cni-plugin
    jobs:
    - name: "cni-plugin tests"
      commands:
      - ../.semaphore/run-and-monitor ci.log make ci

- name: 'calicoq'
  run:
    when: "change_in(['/*', '/calicoq/', '/libcalico-go/'], {exclude: ['/**/.gitignore', '/**/README.md', '/**/LICENSE']})"
  dependencies: ["Prerequisites"]
  task:
    secrets:
    - name: test-customer-license
    prologue:
      commands:
      - cd calicoq
    jobs:
    - name: "make ci"
      commands:
      - ../.semaphore/run-and-monitor ci.log make ci

- name: 'anomaly-detection-api'
  run:
    when: "change_in(['/*', '/anomaly-detection-api/'], {exclude: ['/**/.gitignore', '/**/README.md', '/**/LICENSE']})"
  dependencies: ["Prerequisites"]
  task:
    secrets:
    - name: test-customer-license
    prologue:
      commands:
      - cd anomaly-detection-api
    jobs:
    - name: "make ci"
      commands:
      - ../.semaphore/run-and-monitor ci.log make ci

- name: 'compliance'
  run:
    when: "${FORCE_RUN} or change_in(['/*', '/compliance/', '/api/', '/libcalico-go/', '/lma/'], {exclude: ['/**/.gitignore', '/**/README.md', '/**/LICENSE']})"
  dependencies: ["Prerequisites"]
  task:
    secrets:
    - name: test-customer-license
    prologue:
      commands:
      - cd compliance
    jobs:
    - name: "compliance tests"
      commands:
      - ../.semaphore/run-and-monitor ci.log make ci

- name: 'continuous-policy-recommendation'
  run:
    when: "${FORCE_RUN} or change_in(['/*', '/continuous-policy-recommendation/', '/api/', '/libcalico-go/', '/lma/'], {exclude: ['/**/.gitignore', '/**/README.md', '/**/LICENSE']})"
  dependencies: ["Prerequisites"]
  task:
    secrets:
    - name: test-customer-license
    prologue:
      commands:
      - cd continuous-policy-recommendation
    jobs:
    - name: "continuous-policy-recommendation tests"
      commands:
      - ../.semaphore/run-and-monitor ci.log make ci

- name: 'deep-packet-inspection'
  run:
    when: "${FORCE_RUN} or change_in(['/*', '/deep-packet-inspection/', '/api/', '/libcalico-go/', '/lma/', '/typha/'], {exclude: ['/**/.gitignore', '/**/README.md', '/**/LICENSE']})"
  dependencies: ["Prerequisites"]
  task:
    secrets:
    - name: test-customer-license
    prologue:
      commands:
      - cd deep-packet-inspection
    jobs:
    - name: "deep-packet-inspection tests"
      commands:
      - ../.semaphore/run-and-monitor ci.log make ci

- name: "egress-gateway"
  run:
    when: "${FORCE_RUN} or change_in(['/*', '/egress-gateway/', '/libcalico-go/lib/logutils/', '/libcalico-go/lib/health/', '/felix/proto/'], {exclude: ['/**/.gitignore', '/**/README.md', '/**/LICENSE']})"
  dependencies: ["Prerequisites"]
  task:
    prologue:
      commands:
      - cd egress-gateway
    jobs:
      - name: "make ci"
        commands:
         - ../.semaphore/run-and-monitor ci.log make ci

- name: 'es-gateway'
  run:
    when: "${FORCE_RUN} or change_in(['/*', '/es-gateway/', '/libcalico-go/lib/logutils/'], {exclude: ['/**/.gitignore', '/**/README.md', '/**/LICENSE']})"
  dependencies: ["Prerequisites"]
  task:
    secrets:
    - name: test-customer-license
    prologue:
      commands:
      - cd es-gateway
    jobs:
    - name: "es-gateway tests"
      commands:
      - ../.semaphore/run-and-monitor ci.log make ci

- name: 'es-proxy'
  run:
    when: "${FORCE_RUN} or change_in(['/*', '/es-proxy/', '/api/', '/compliance/', '/libcalico-go/', '/lma/'], {exclude: ['/**/.gitignore', '/**/README.md', '/**/LICENSE']})"
  dependencies: ["Prerequisites"]
  task:
    secrets:
    - name: test-customer-license
    prologue:
      commands:
      - cd es-proxy
    jobs:
    - name: "es-proxy tests"
      commands:
      - ../.semaphore/run-and-monitor ci.log make ci

- name: 'firewall-integration'
  run:
    when: "${FORCE_RUN} or change_in(['/*', '/firewall-integration/', '/api/', '/compliance/', '/kube-controllers/', '/libcalico-go/', '/lma/', '/felix/'], {exclude: ['/**/.gitignore', '/**/README.md', '/**/LICENSE']})"
  dependencies: ["Prerequisites"]
  task:
    secrets:
    - name: test-customer-license
    prologue:
      commands:
      - cd firewall-integration
    jobs:
    - name: "firewall-integration tests"
      commands:
      - ../.semaphore/run-and-monitor ci.log make ci

- name: 'honeypod-controller'
  run:
    when: "${FORCE_RUN} or change_in(['/*', '/honeypod-controller/', '/libcalico-go/', '/licensing/', '/lma/'], {exclude: ['/**/.gitignore', '/**/README.md', '/**/LICENSE']})"
  dependencies: ["Prerequisites"]
  task:
    secrets:
    - name: test-customer-license
    prologue:
      commands:
      - cd honeypod-controller
    jobs:
    - name: "honeypod-controller tests"
      commands:
      - ../.semaphore/run-and-monitor ci.log make ci

- name: 'intrusion-detection-controller'
  run:
    when: "${FORCE_RUN} or change_in(['/*', '/intrusion-detection-controller/', '/api/', '/libcalico-go/', '/licensing/', '/lma/'], {exclude: ['/**/.gitignore', '/**/README.md', '/**/LICENSE']})"
  dependencies: ["Prerequisites"]
  task:
    secrets:
    - name: test-customer-license
    prologue:
      commands:
      - cd intrusion-detection-controller
    jobs:
    - name: "intrusion-detection-controller tests"
      commands:
      - ../.semaphore/run-and-monitor ci.log make ci

- name: "l7-collector"
  run:
    when: "${FORCE_RUN} or change_in(['/*', '/l7-collector/', '/libcalico-go/lib/testutils'], {exclude: ['/**/.gitignore', '/**/README.md', '/**/LICENSE']})"
  dependencies: ["Prerequisites"]
  task:
    prologue:
      commands:
      - cd l7-collector
    jobs:
    - name: "l7-collector tests"
      commands:
      - ../.semaphore/run-and-monitor ci.log make ci

- name: "licensing"
  run:
    when: "${FORCE_RUN} or change_in(['/*', '/licensing/', '/libcalico-go/'], {exclude: ['/**/.gitignore', '/**/README.md', '/**/LICENSE']})"
  dependencies: ["Prerequisites"]
  task:
    prologue:
      commands:
      - cd licensing
    jobs:
    - name: "Licensing tests"
      commands:
      - ../.semaphore/run-and-monitor ci.log make ci

- name: "license-agent"
  run:
    when: "${FORCE_RUN} or change_in(['/*', '/licensing/', '/libcalico-go/'], {exclude: ['/**/.gitignore', '/**/README.md', '/**/LICENSE']})"
  dependencies: ["Prerequisites"]
  task:
    prologue:
      commands:
      - cd license-agent
    jobs:
    - name: "License Agent tests"
      commands:
      - ../.semaphore/run-and-monitor ci.log make ci

- name: "lma"
  run:
    when: "${FORCE_RUN} or change_in(['/*', '/lma/', '/libcalico-go/'], {exclude: ['/**/.gitignore', '/**/README.md', '/**/LICENSE']})"
  dependencies: ["Prerequisites"]
  task:
    prologue:
      commands:
      - cd lma
    jobs:
    - name: "lma tests"
      commands:
      - ../.semaphore/run-and-monitor ci.log make ci

- name: 'packetcapture'
  run:
    when: "${FORCE_RUN} or change_in(['/*', '/packetcapture', '/api/', '/libcalico-go/', '/lma/'], {exclude: ['/**/.gitignore', '/**/README.md', '/**/LICENSE']})"
  dependencies: ["Prerequisites"]
  task:
    secrets:
    - name: test-customer-license
    prologue:
      commands:
      - cd packetcapture
    jobs:
    - name: "packetcapture tests"
      commands:
      - ../.semaphore/run-and-monitor ci.log make ci

- name: 'prometheus-service'
  run:
    when: "${FORCE_RUN} or change_in(['/*', '/prometheus-service/', 'libcalico-go/lib/logutils/', '/lma/'], {exclude: ['/**/.gitignore', '/**/README.md', '/**/LICENSE']})"
  dependencies: ["Prerequisites"]
  task:
    secrets:
    - name: test-customer-license
    prologue:
      commands:
      - cd prometheus-service
    jobs:
    - name: "prometheus-service tests"
      commands:
      - make ci

- name: 'ts-queryserver'
  run:
    when: "${FORCE_RUN} or change_in(['/*', '/ts-queryserver/', '/api/', '/calicoctl/', '/felix/', '/lma/', '/libcalico-go/', '/licensing/'], {exclude: ['/**/.gitignore', '/**/README.md', '/**/LICENSE']})"
  dependencies: ["Prerequisites"]
  task:
    secrets:
    - name: test-customer-license
    prologue:
      commands:
      - cd ts-queryserver
    jobs:
    - name: "ts-queryserver tests"
      commands:
      - ../.semaphore/run-and-monitor ci.log make ci

- name: 'voltron'
  run:
    when: "${FORCE_RUN} or change_in(['/*', '/voltron/', '/api/', '/libcalico-go/lib/logutils/', '/lma/'], {exclude: ['/**/.gitignore', '/**/README.md', '/**/LICENSE']})"
  dependencies: ["Prerequisites"]
  task:
    secrets:
    - name: test-customer-license
    prologue:
      commands:
      - cd voltron
    jobs:
    - name: "voltron tests"
      commands:
      - ../.semaphore/run-and-monitor ci.log make ci

- name: "Documentation"
  run:
    when: "${FORCE_RUN} or change_in(['/*', '/calico/'], {exclude: ['/**/.gitignore', '/**/README.md', '/**/LICENSE']})"
  dependencies: ["Prerequisites"]
  task:
    prologue:
      commands:
      - cd calico
    jobs:
    - name: "htmlproofer, kubeval"
      commands:
      - ../.semaphore/run-and-monitor htmlproofer.log make htmlproofer
      - ../.semaphore/run-and-monitor kubeval.log make kubeval

after_pipeline:
  task:
    jobs:
    - name: Reports
      commands:
        - test-results gen-pipeline-report --force
