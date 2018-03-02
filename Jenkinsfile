#!groovy
pipeline {
    agent { label 'slave-large'}
    triggers{
        pollSCM('H/5 * * * *')
        cron('H H(0-7) * * 1-5')
    }
    stages {
        stage('Checkout') {
            steps {
                checkout scm
                script {
                    currentBuild.description = """
                    BRANCH_NAME=${env.BRANCH_NAME}
                    JOB_NAME=${env.JOB_NAME}
                    BUILD_INFO=${env.RUN_DISPLAY_URL}""".stripIndent()
                }
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
                withCredentials([sshUserPrivateKey(credentialsId: 'marvin-tigera-ssh-key', keyFileVariable: 'SSH_KEY', passphraseVariable: '', usernameVariable: '')]) {
                    // Needed to allow checkout of private repos
                    ansiColor('xterm') {
                        echo 'make test-containerized..'
                        sh 'if [ -z "$SSH_AUTH_SOCK" ] ; then eval `ssh-agent -s`; ssh-add SSH_KEY || true; fi && make vendor static-checks test-containerized'
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
                slackSend message: "Failure during ${env.JOB_NAME} CI!\n${env.RUN_DISPLAY_URL}", color: "warning", channel: "cnx-ci-failures"
              }
            }
        }
    }
}
