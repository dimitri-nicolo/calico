#!groovy
pipeline {
    agent { label 'slave'}
    options{
        timeout(time: 2, unit: 'HOURS')
    }
    triggers{
        pollSCM('H/5 * * * *')
        cron('H H(0-7) * * 1-5')
    }
    environment {
        IMAGE_NAME = "gcr.io/unique-caldron-775/cnx/tigera/cni"
        WAVETANK_SERVICE_ACCT = "wavetank@unique-caldron-775.iam.gserviceaccount.com"
        AWS_REGION="us-west-2"

        NAME_PREFIX="wt-winfv-${env.BRANCH_NAME}-${env.BUILD_NUMBER}"
        KUBE_VERSION = "1.11.2"
        WINDOWS_KEYPAIR_NAME="wavetank"
        WINDOWS_PEM_FILE="/home/jenkins/.ssh/wavetank.pem"
        WINDOWS_PPK_FILE="/home/jenkins/.ssh/wavetank.ppk"
        RDP_SOURCE_CIDR="0.0.0.0/0"
        FV_TIMEOUT="1800"
    }
    stages {
        stage('Checkout') {
            steps {
                script {
                    checkout scm

                    currentBuild.description = """
                    BRANCH_NAME=${env.BRANCH_NAME}
                    JOB_NAME=${env.JOB_NAME}
                    IMAGE_NAME=${env.IMAGE_NAME}:${env.BRANCH_NAME}
                    BUILD_INFO=${env.RUN_DISPLAY_URL}""".stripIndent()

                    echo "Checkout complete. Start to run windows FV."
                    env.RUN_WINDOWS_FV = "YES"
                }
            }
        }
        stage('Build windows binary') {
            when {
                expression { env.RUN_WINDOWS_FV == "YES" }
            }
            steps {
                withCredentials([sshUserPrivateKey(credentialsId: 'marvin-tigera-ssh-key', keyFileVariable: 'SSH_KEY', passphraseVariable: '', usernameVariable: '')]) {
                    ansiColor('xterm') {
                        // Needed to allow checkout of private repos
                        sh 'if [ -z "$SSH_AUTH_SOCK" ] ; then eval `ssh-agent -s`; ssh-add $SSH_KEY || true; fi && make build && make bin/amd64/win-fv.exe'
                    }
                }
            }
        }
        stage('Initialization') {
            when {
                expression { env.RUN_WINDOWS_FV == "YES" }
            }
            steps {
                script {
                    withCredentials([file(credentialsId: 'aws-credentials-key', variable: 'AWS_CREDS'),
                        file(credentialsId: 'registry-viewer-account-json', variable: 'DOCKER_AUTH'),
                        file(credentialsId: 'aws-wavetank-ssh-keypair', variable: 'KEY'),
                        file(credentialsId: 'aws-wavetank-ssh-keypair-pub', variable: 'PUB')]) {
                        sh '''#!/bin/bash
                            set -x
                            # prep secrets
                            mkdir -p ~/.aws ~/.ssh
                            cp ${AWS_CREDS} ~/.aws/credentials
                            echo "[default]" > ~/.aws/config
                            echo "region = ${AWS_REGION}" >> ~/.aws/config
                            cp ${KEY} ~/.ssh/wavetank.pem
                            cp ${PUB} ~/.ssh/wavetank.pub

                            sudo apt-get install putty-tools
                            puttygen ~/.ssh/wavetank.pem -O private -o ~/.ssh/wavetank.ppk
                            chmod 600 ~/.ssh/*
                            ls -ltr ~/.ssh/
                            cp $DOCKER_AUTH ~/docker_auth.json
                            echo pemfile : ${WINDOWS_PEM_FILE}"
                            echo ppkfile ${WINDOWS_PPK_FILE}"
                        '''
                        AWS_ACCESS_KEY_ID = sh (returnStdout: true, script: 'aws configure get aws_access_key_id').trim()
                        AWS_SECRET_ACCESS_KEY = sh (returnStdout: true, script: 'aws configure get aws_secret_access_key').trim()
                    }
                }
            }
        }

        stage('Checkout process') {
            when {
                expression { env.RUN_WINDOWS_FV == "YES" }
            }
            steps {
                dir('process') {
                    git(url: 'git@github.com:tigera/process.git', branch: 'master', credentialsId: 'marvin-tigera-ssh-key')
                }
            }
        }

        stage('Create FV cluster') {
            when {
                expression { env.RUN_WINDOWS_FV == "YES" }
            }
            steps {
                dir('process/testing/winfv') {
                    withCredentials([file(credentialsId: 'cnx-license-key', variable: 'KEY')]) {
                         sh "cp $KEY license.yaml"
                         sh "cp ~/docker_auth.json docker_auth.json"
                         sh "cp ${env.WORKSPACE}/internal/pkg/testutils/private.key private.key"
                         sh "cp ${env.WORKSPACE}/bin/amd64/*.exe ."
                         sh "echo pemfile : ${WINDOWS_PEM_FILE}"
                         sh "echo ppkfile ${WINDOWS_PPK_FILE}"
                         sh "./setup-fv.sh -q"
                    }
                }
            }
        }

        stage('Run Windows FV\'s') {
            when {
                expression { env.RUN_WINDOWS_FV == "YES" }
            }
            steps {
                 ansiColor('xterm') {
                     dir('process/testing/winfv') {
                             sh '''
                                #!/bin/bash
                                MASTER_IP=$(aws ec2 describe-instances --filters Name=tag-value,Values=win-${NAME_PREFIX}-fv-linux --query "Reservations[0].Instances[0].PublicIpAddress" --output text)
                                SSH_CMD=$(echo ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -i ~/.ssh/wavetank.pem ubuntu@${MASTER_IP})
                                SCP_CMD=$(echo scp -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -i ~/.ssh/wavetank.pem)

                                ${SSH_CMD} ls -ltr /home/ubuntu
                                ${SSH_CMD} ls -ltr /home/ubuntu/winfv
                                ${SSH_CMD} touch /home/ubuntu/file-ready
                                ${SSH_CMD} time /home/ubuntu/winfv/wait-report.sh
                                ${SSH_CMD} ls -ltr /home/ubuntu/report
                             '''
                     }
                 }
            }
        }
    }
    post {
        success {
            echo "Yay, we passed."
        }

        always {
            script {
                if (env.RUN_WINDOWS_FV != "YES" ) {
                    return
                }

                sh '''
                    #!/bin/bash
                    MASTER_IP=$(aws ec2 describe-instances --filters Name=tag-value,Values=win-${NAME_PREFIX}-fv-linux --query "Reservations[0].Instances[0].PublicIpAddress" --output text)
                    SSH_CMD=$(echo ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -i ~/.ssh/wavetank.pem ubuntu@${MASTER_IP})
                    SCP_CMD=$(echo scp -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -i ~/.ssh/wavetank.pem)
                    ${SSH_CMD} bash -c \\"sudo chown -R ubuntu:ubuntu *\\" || true
                    ${SCP_CMD} -r ubuntu@${MASTER_IP}:/home/ubuntu/report . || true
                    ls -ltr .
                    ls -ltr ./report
                '''
                junit allowEmptyResults: true, testResults: 'report/*.xml'
                dir('process/testing/winfv') {
                    sh '''
                       #!/bin/bash
                       ./setup-fv.sh -u -q
                    '''
               }
            }
        }

        changed { // Notify only on change to success
            script {
                if (env.BRANCH_NAME ==~ /(master|release-.*)/) {
                    GIT_HASH = env.GIT_COMMIT[0..6]
                    GIT_AUTHOR = sh(returnStdout: true, script: "git show -s --format='%an' ${env.GIT_COMMIT}").trim()
                    if (currentBuild.currentResult == 'SUCCESS' && currentBuild.getPreviousBuild()?.result) {
                        msg = "Passing again ${env.JOB_NAME}\n${GIT_AUTHOR} ${GIT_HASH}\n${env.RUN_DISPLAY_URL}"
                        slackSend message: msg, color: "good", channel: "ci-notifications-song"
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
                    slackSend message: msg, color: "danger", channel: "ci-notifications-song"
                }
            }
        }
    }
}
