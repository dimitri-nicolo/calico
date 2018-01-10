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
                echo 'Clean artifacts..'
                sh 'if [ -z "$SSH_AUTH_SOCK" ] ; then eval `ssh-agent -s`; ssh-add || true; fi && make clean'

            }
        }
        stage('make test-containerized') {
            steps {
                ansiColor('xterm') {
                    // Needed to allow checkout of private repos
                    echo 'make test-containerized..'
                    sh 'if [ -z "$SSH_AUTH_SOCK" ] ; then eval `ssh-agent -s`; ssh-add || true; fi && make vendor test-containerized'
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
                slackSend message: "Failure during libcalico-go-private master CI!\nhttp://localhost:8080/view/Essentials/job/Tigera/job/libcalico-go-private/job/master/", color: "warning", channel: "cnx-ci-failures"
              }
            }
        }
    }
}
