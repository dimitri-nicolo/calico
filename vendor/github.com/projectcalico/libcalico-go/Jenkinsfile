#!groovy
pipeline {
    agent { label 'slave'}
    triggers{
        pollSCM('H/5 * * * *')
        cron('H H(0-7) * * *')
    }
    environment {
        BRANCH_NAME = 'master'
        BUILD_INFO = "https://wavetank.tigera.io/blue/organizations/jenkins/${env.JOB_NAME}/detail/${env.JOB_NAME}/${env.BUILD_NUMBER}/pipeline"
        SLACK_MSG = "Failure during ${env.JOB_NAME}:${env.BRANCH_NAME} CI!\n${env.BUILD_INFO}"
    }
    stages {
        stage('Checkout') {
            steps {
                script {
                    currentBuild.description = """
                    BRANCH_NAME=${env.BRANCH_NAME}
                    JOB_NAME=${env.JOB_NAME}
                    BUILD_INFO=${env.BUILD_INFO}""".stripIndent()
                }
                git(url: 'git@github.com:tigera/libcalico-go-private.git', credentialsId: 'marvin-tigera-ssh-key', branch: "${env.BRANCH_NAME}")
            }
        }
        stage('make test-containerized') {
            steps {
                withCredentials([sshUserPrivateKey(credentialsId: 'marvin-tigera-ssh-key', keyFileVariable: 'SSH_KEY', passphraseVariable: '', usernameVariable: '')]) {
                    // Needed to allow checkout of private repos
                    ansiColor('xterm') {
                        echo 'make test-containerized..'
                        sh 'if [ -z "$SSH_AUTH_SOCK" ] ; then eval `ssh-agent -s`; ssh-add SSH_KEY || true; fi && make vendor test-containerized'
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
                slackSend message: "${env.SLACK_MSG}", color: "warning", channel: "cnx-ci-failures"
              }
            }
        }
    }
}
