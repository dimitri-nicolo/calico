#!groovy
pipeline {
    agent { label 'slave'}
    options{
        timeout(time: 2, unit: 'HOURS')
        buildDiscarder(logRotator(numToKeepStr: '30', artifactNumToKeepStr: '30'))
    }
    triggers{
        pollSCM('H/5 * * * *')
    }
    environment {
        WAVETANK_SERVICE_ACCT = "wavetank@unique-caldron-775.iam.gserviceaccount.com"

        // Jenkins job info
        SANE_JOB_NAME =  "${env.JOB_BASE_NAME}".replace('.', '-').toLowerCase()

        // Terraform base settings
        TF_VAR_google_project = 'unique-caldron-775'
        TF_VAR_google_region = 'us-central1'
        TF_VAR_zone = 'us-central1-f'
        TF_VAR_prefix = "wt-${SANE_JOB_NAME}-${env.BUILD_NUMBER}"
        TF_VAR_node_preemptible = 'false'
        TF_VAR_master_disk_size = '100'
        TF_VAR_num_etcd_nodes='0'
        TF_VAR_num_elastic_nodes = '0'

        // Test params
        MANIFEST = 'https://docs.tigera.io/master/getting-started/kubernetes/installation/hosted/calico.yaml'
        RBAC_MANIFEST = ''
        E2E_IMAGE = 'gcr.io/unique-caldron-775/k8s-e2e:master'
        MANAGER_MANIFEST = ''
        E2E_FLAGS = '--extended-networking true --calico-version v3  --cnx v3 --skip "Feature:CNX-v3-RBAC-PerTier" --extra-args " --calicoctl-image=gcr.io/unique-caldron-775/cnx/tigera/calicoctl:master"'

        // install-cnx.sh settings
        DATASTORE = "etcdv3"

        // Slack message params
        BUILD_INFO = "https://wavetank.tigera.io/blue/organizations/jenkins/${env.JOB_NAME}/detail/${env.JOB_NAME}/${env.BUILD_NUMBER}/pipeline"
        SLACK_MSG = "${BUILD_INFO}\n- *k8s-version:* ${TF_VAR_kubernetes_version}\n"
        slack_alert_channel = 'ci-notifications-cnx'
    }
    stages {        
        
        stage('Initialize') {
            steps {
                dir('process') {
                    git(url: 'git@github.com:tigera/process.git', branch: 'master', credentialsId: 'marvin-tigera-ssh-key')
                    script {
                        env.TF_VAR_kubernetes_version = sh (returnStdout: true, script: "./k8s_version.sh stable-1.13").trim()
                        if ( "$env.TF_VAR_kubernetes_version" == "[FAIL] did not resolve a k8s version" ){
                            currentBuild.result = "UNSTABLE"
                            echo "Current build status: ${currentBuild.result}"
                            return
                            }
                    }
                }
                script {
                    currentBuild.description = """\
                    K8S_RELEASE=${env.TF_VAR_kubernetes_version}
                    E2E_IMAGE=${env.E2E_IMAGE}
                    E2E_FLAGS=${env.E2E_FLAGS}
                    RBAC_MANIFEST=${env.RBAC_MANIFEST}
                    BUILD_INFO=${env.BUILD_INFO}""".stripIndent()                    
                }
            }
        }
        stage('Clean artifacts') {
            steps {
                echo 'clean artifacts..'
                sh 'if [ -z "$SSH_AUTH_SOCK" ] ; then eval `ssh-agent -s`; ssh-add || true; fi && make clean'
            }
        }
        stage('Check calico-private changes') {
            steps {
                script {
                    SKIP_CALICO_PRIVATE_MASTER_BUILD = sh(returnStatus: true, script: "git diff --name-only HEAD^ | grep '^_includes/master/manifests/\\|^master/getting-started/kubernetes/install-cnx.sh\\|^master/getting-started/kubernetes/installation/hosted/'")
                    echo "SKIP_CALICO_PRIVATE_MASTER_BUILD=${SKIP_CALICO_PRIVATE_MASTER_BUILD}"
                }
            }
        }
        stage('Provision Cluster') {
            when {
                expression { SKIP_CALICO_PRIVATE_MASTER_BUILD == 0 && currentBuild.result != "UNSTABLE" }
            }
            steps {
                dir('crc') {
                    git(url: 'git@github.com:tigera/calico-ready-clusters.git', branch: 'master', credentialsId: 'marvin-tigera-ssh-key')
                    dir('kubeadm/1.6/') {
                        withCredentials([file(credentialsId: 'registry-viewer-account-json', variable: 'DOCKER_AUTH')]) {
                            sh "cp $DOCKER_AUTH docker_auth.json"
                            sh "terraform init -upgrade"
                            sh "terraform apply -var num_etcd_nodes=${TF_VAR_num_etcd_nodes}"
                            script {                              
                              // the self-hosted etcd install requires an explicit value
                              env.ETCD_ENDPOINTS = "http://10.96.232.136:6666"
                            }

                            // Remove the taint from master, but expect an error as this command tries to remove the taint from nodes (which do not have it).
                            sh '$(terraform output master_connect_command) /usr/bin/kubectl taint nodes --all node-role.kubernetes.io/master- | true'
                        }
                    }
                }
            }
            post {
                failure {
                    slackSend message: "Failed during Terraform provisioning!'\n${env.SLACK_MSG}", color: "warning", channel: "${env.slack_alert_channel}"
                }
            }
        }
        stage('Checkout calico-private') {
            when {
                expression { SKIP_CALICO_PRIVATE_MASTER_BUILD == 0 && currentBuild.result != "UNSTABLE" }
            }
            steps {
                sh "mkdir ~/calico-private && cp -R ./ ~/calico-private/"
                dir('crc/kubeadm/1.6') {
                    sh "scp -i master_ssh_key -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no -r ~/calico-private/ \$(terraform output master_connect):~/"
                }
            }
        }        
        stage('Build calico-private') {
            when {
                expression { SKIP_CALICO_PRIVATE_MASTER_BUILD == 0 && currentBuild.result != "UNSTABLE" }
            }
            steps {
                withCredentials([sshUserPrivateKey(credentialsId: 'marvin-tigera-ssh-key', keyFileVariable: 'SSH_KEY', passphraseVariable: '', usernameVariable: '')]) {
                    ansiColor('xterm') {
                        dir('crc/kubeadm/1.6') {
                            // Needed to allow checkout of private repos
                            echo 'Build calico-private...'
                            sh '$(terraform output master_connect_command) "sudo apt-get -y update && sudo apt-get -y install build-essential"'
                            sh '$(terraform output master_connect_command) "cd ~/calico-private/ && make _site && make ci"'
                            sh '$(terraform output master_connect_command) "cp ~/calico-private/_site/master/getting-started/kubernetes/install-cnx.sh ~/"'
                            sh '$(terraform output master_connect_command) "cp ~/calico-private/_site/master/getting-started/kubernetes/installation/hosted/calico.yaml ~/"'
                            sh '$(terraform output master_connect_command) "cp ~/calico-private/_site/master/getting-started/kubernetes/installation/hosted/etcd.yaml ~/"'
                            sh '$(terraform output master_connect_command) "cp ~/calico-private/_site/master/getting-started/kubernetes/installation/hosted/calicoctl.yaml ~/"'
                            sh '$(terraform output master_connect_command) "cp ~/calico-private/_site/master/getting-started/kubernetes/installation/hosted/calicoq.yaml ~/"'
                            sh '$(terraform output master_connect_command) "cp ~/calico-private/_site/master/getting-started/kubernetes/installation/hosted/cnx/1.7/cnx-policy.yaml ~/"'
                            sh '$(terraform output master_connect_command) "cp ~/calico-private/_site/master/getting-started/kubernetes/installation/hosted/cnx/1.7/operator.yaml ~/"'
                            sh '$(terraform output master_connect_command) "cp ~/calico-private/_site/master/getting-started/kubernetes/installation/hosted/cnx/1.7/monitor-calico.yaml ~/"'
                            sh '$(terraform output master_connect_command) "cp ~/calico-private/_site/master/getting-started/kubernetes/installation/hosted/cnx/1.7/cnx-api-etcd.yaml ~/"'
                        }
                    }
                }
            }
        }
        stage('Install CNX') {
        when {
                expression { SKIP_CALICO_PRIVATE_MASTER_BUILD == 0 && currentBuild.result != "UNSTABLE" }
            }
            steps {
                dir('crc/kubeadm/1.6') {
                    // Create docker auth file to allow pulls from quay.io/tigera
                    withCredentials([file(credentialsId: 'quay-read-json', variable: 'KEY')]) {
                        sh "cp $KEY config.json"
                        sh "scp -i master_ssh_key -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no config.json \$(terraform output master_connect):~/config.json"
                    }

                    withCredentials([file(credentialsId: 'wavetank_service_account', variable: 'DOCKER_AUTH')]) {
                        sh "gcloud auth activate-service-account wavetank@unique-caldron-775.iam.gserviceaccount.com --key-file $DOCKER_AUTH"
                        sh "gcloud docker --authorize-only --server gcr.io"
                    }

                    // Create a CNX license file
                    withCredentials([file(credentialsId: 'cnx-license-key', variable: 'KEY')]) {
                        sh "cp $KEY license.yaml"
                        sh "scp -i master_ssh_key -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no license.yaml \$(terraform output master_connect):~/license.yaml"
                    }

                    sh '$(terraform output master_connect_command) "chmod +x ./install-cnx.sh"'
                    sh '$(terraform output master_connect_command) "DATASTORE=$DATASTORE ./install-cnx.sh -q"'
                }
            }
            post {
                failure {
                    slackSend message: "Failed to install CNX with quick-staller!'\n${env.SLACK_MSG}", color: "warning", channel: "${env.slack_alert_channel}"
                }
            }
        }
        stage('Run e2e\'s') {
        when {
                expression { SKIP_CALICO_PRIVATE_MASTER_BUILD == 0 && currentBuild.result != "UNSTABLE" }
            }
            steps {
                ansiColor('xterm') {
                    dir('crc/kubeadm/1.6') {
                        sh '$(terraform output master_connect_command) docker run -t -v /home/ubuntu/report:/report -v /home/ubuntu/.kube/config:/root/kubeconfig -v /usr/bin/kubectl:/usr/bin/kubectl --net=host $E2E_IMAGE $E2E_FLAGS'
                    }
                }
            }
            post {
                failure {
                   slackSend message: "e2e tests failed!\n${env.RUN_DISPLAY_URL}\n- *Manifest:* `${env.MANIFEST}`\n- *e2e-image:* ${env.E2E_IMAGE}", color: "warning", channel: "${env.slack_alert_channel}"
                }
                always {
                    dir('crc/kubeadm/1.6') {
                        sh 'scp -r -i ./master_ssh_key -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no $(terraform output master_connect):/home/ubuntu/report . | true'
                        junit allowEmptyResults: true, testResults: 'report/*.xml'
                    }
                }
                changed { // Notify only on change to success
                    script {
                        if (currentBuild.currentResult == 'SUCCESS' && currentBuild.getPreviousBuild()?.result) {
                            msg = "Passing again ${env.JOB_NAME}\n${env.RUN_DISPLAY_URL}"
                            slackSend message: msg, color: "good", channel: "${env.slack_alert_channel}"
                        }
                    }
                }
            }
        }
    }
    post {
        always {
            script{
            if( SKIP_CALICO_PRIVATE_MASTER_BUILD == 0 && currentBuild.result  != "UNSTABLE" ){
                        dir('crc/kubeadm/1.6') {
                            echo '****Getting all kube-system pods****'
                            sh '$(terraform output master_connect_command) "kubectl get po -n kube-system -o wide"'
                            echo '****Getting describe for calico-node pods****'
                            sh '$(terraform output master_connect_command) "kubectl describe po -n kube-system -l k8s-app=calico-node"'
                            echo '****Getting calico node logs****'
                            sh '$(terraform output master_connect_command) "kubectl logs -n kube-system -l k8s-app=calico-node --container=calico-node"'
                            echo '****Getting describe for cnx-manager****'
                            sh '$(terraform output master_connect_command) "kubectl describe po -n kube-system -l k8s-app=cnx-manager"'
                            echo '****Getting logs for cnx-manager****'
                            sh '$(terraform output master_connect_command) "kubectl logs -n kube-system -l k8s-app=cnx-manager --container=cnx-manager"'
                            echo '****Getting list of all pods****'
                            sh '$(terraform output master_connect_command) "kubectl get po --all-namespaces"'
                            echo '****Getting list of all services****'
                            sh '$(terraform output master_connect_command) "kubectl get svc --all-namespaces"'
                            echo '****Diags done.  Destroying****'
                            sh 'terraform destroy -force'
                    }
                }
            }
        }
    }
}