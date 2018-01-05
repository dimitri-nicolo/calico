#!groovy
pipeline {
    agent { label 'containers'}
    triggers{
        pollSCM('H/5 * * * *')
        cron('H H(0-7) * * *')
    }

    stages {
        stage('Checkout') {
            steps {
                echo 'checkout scm..'
                checkout scm
            }
        }
        stage('Clean artifacts') {
            steps {
                echo 'clean artifacts..'
                sh 'if [ -z "$SSH_AUTH_SOCK" ] ; then eval `ssh-agent -s`; ssh-add || true; fi && make clean'
            }
        }
        stage('Build apiserver') {
            steps {
                ansiColor('xterm') {
                    // Needed to allow checkout of private repos
                    echo 'Build apiserver..'
                    sh 'if [ -z "$SSH_AUTH_SOCK" ] ; then eval `ssh-agent -s`; ssh-add || true; fi && make all'
                }
            }
        }
        stage('Test') {
            steps {
                echo 'Testing ut fv..'
                sh 'if [ -z "$SSH_AUTH_SOCK" ] ; then eval `ssh-agent -s`; ssh-add || true; fi && make test'
            }
        }
        stage('Push image to GCR') {
            steps {
                script{
                    // Will eventually want to only push for passing builds. Cannot for now since the builds don't all pass currently
                    // if (env.BRANCH_NAME == 'master' && (currentBuild.result == null || currentBuild.result == 'SUCCESS')) {
                    if (env.BRANCH_NAME == 'master') {
                        sh 'docker tag tigera/cnx-apiserver:latest gcr.io/tigera-dev/cnx/tigera/cnx-apiserver:master'
                        sh 'gcloud docker -- push gcr.io/tigera-dev/cnx/tigera/cnx-apiserver:master'

                        // Clean up images.
                        // Hackey since empty displayed tags are not empty according to gcloud filter criteria
                        sh '''for digest in $(gcloud container images list-tags gcr.io/tigera-dev/cnx/tigera/cnx-apiserver --format='get(digest)'); do
                                if ! test $(echo $(gcloud container images list-tags gcr.io/tigera-dev/cnx/tigera/cnx-apiserver --filter=digest~${digest}) | awk '{print $6}'); then
                                    gcloud container images delete -q --force-delete-tags "gcr.io/tigera-dev/cnx/tigera/cnx-apiserver@${digest}"
                                fi
                        done'''
                    }
                }
            }
        }
    }
    post {
        success {
            echo "Yay, we passed."
        }
        failure {
            echo "Boo, we failed."
            script {
                if (env.BRANCH_NAME == 'master') {
                    slackSend message: "Failure during calico-k8sapiserver:master CI!\nhttp://localhost:8080/view/Essentials/job/Tigera/job/calico-k8sapiserver/job/master/", color: "warning", channel: "cnx-ci-failures"
                }
            }
        }
    }
}
