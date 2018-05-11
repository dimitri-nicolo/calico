#!groovy
pipeline {
    options {
        disableConcurrentBuilds()
    }
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
                sh 'if [ -z "$SSH_AUTH_SOCK" ] ; then eval `ssh-agent -s`; ssh-add || true; fi && make clean'

            }
        }
        stage('make test-containerized') {
            steps {
                withCredentials([sshUserPrivateKey(credentialsId: 'marvin-tigera-ssh-key', keyFileVariable: 'SSH_KEY', passphraseVariable: '', usernameVariable: '')]) {
                    // Needed to allow checkout of private repos
                    ansiColor('xterm') {
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
        changed { // Notify only on change to success
            script {
                if (env.BRANCH_NAME ==~ /(master|release-.*)/) {
                    GIT_HASH = env.GIT_COMMIT[0..6]
                    GIT_AUTHOR = sh(returnStdout: true, script: "git show -s --format='%an' ${env.GIT_COMMIT}").trim()
                    if (currentBuild.currentResult == 'SUCCESS' && currentBuild.getPreviousBuild()?.result) {
                        msg = "Passing again ${env.JOB_NAME}\n${GIT_AUTHOR} ${GIT_HASH}\n${env.RUN_DISPLAY_URL}"
                        slackSend message: msg, color: "good", channel: "ci-notifications-cnx"
                    }
                }
           }
        }
        failure {
            echo "Boo, we failed."
            script {
                if (env.BRANCH_NAME ==~ /(master|release-.*)/) {
                    GIT_HASH = env.GIT_COMMIT[0..6]
                    GIT_AUTHOR = sh(returnStdout: true, script: "git show -s --format='%an' ${env.GIT_COMMIT}").trim()
                    if (currentBuild.getPreviousBuild()?.result == 'FAILURE') {
                        msg = "Still failing ${env.JOB_NAME}\n${GIT_AUTHOR} ${GIT_HASH}\n${env.RUN_DISPLAY_URL}"
                    } else {
                        msg = "New failure ${env.JOB_NAME}\n${GIT_AUTHOR} ${GIT_HASH}\n${env.RUN_DISPLAY_URL}"
                    }
                    slackSend message: msg, color: "danger", channel: "ci-notifications-cnx"
                }
            }
        }
    }
}
