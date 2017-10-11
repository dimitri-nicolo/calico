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
    }
    post {
        success {
            echo "Yay, we passed."
        }
        failure {
            echo "Boo, we failed."
        }
    }
}
