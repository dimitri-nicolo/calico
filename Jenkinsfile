#!groovy
pipeline {
    agent { label 'slave1' }
    triggers{
        pollSCM('H/5 * * * *')
        cron('H H(0-7) * * *')
    }
    stages {
        stage('Checkout') {
            steps {
                checkout scm
            }
        }

        stage('Install Deps') {
            steps {
                ansiColor('xterm') {
                    // Needed to allow checkout of private repos
                    sh 'if [ -z "$SSH_AUTH_SOCK" ] ; then eval `ssh-agent -s`; ssh-add || true; fi && make vendor'
                }
            }
        }
        stage('Build felix') {
            steps {
                sh "echo 'Build Felix'"
                sh "make calico/felix"
            }
        }

        stage('Unit Tests') {
            steps {
                ansiColor('xterm') {
                    sh "echo 'Run unit Tests' && make ut-no-cover"
                }
            }
        }
    }
    post {
        always {
          junit("*/junit.xml")
          deleteDir()
        }
        success {
          echo "Yay, we passed."
        }
        failure {
          echo "Boo, we failed."
        }
    }
}
