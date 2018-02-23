pipeline{
    agent { label 'slave' }
    triggers{
        pollSCM('H/5 * * * *')
        cron('H H(0-7) * * *')
    }
    environment {
        IMAGE_NAME = "gcr.io/unique-caldron-775/cnx/tigera/calicoq"
        WAVETANK_SERVICE_ACCT = "wavetank@unique-caldron-775.iam.gserviceaccount.com"
        BUILD_INFO = "https://wavetank.tigera.io/blue/organizations/jenkins/${env.JOB_NAME}/detail/${env.JOB_NAME}/${env.BUILD_NUMBER}/pipeline"
        SLACK_MSG = "Failure during ${env.JOB_NAME}:${env.BRANCH_NAME} CI!\n${env.BUILD_INFO}"
    }
    stages {
        stage('Checkout') {
            steps {
                checkout scm
                script {
                    currentBuild.description = """
                    BRANCH_NAME=${env.BRANCH_NAME}
                    JOB_NAME=${env.JOB_NAME}
                    IMAGE_NAME=${env.IMAGE_NAME}:${env.BRANCH_NAME}
                    BUILD_INFO=${env.BUILD_INFO}""".stripIndent()
                }
            }
        }
        stage('Clean artifacts') {
            steps {
                sh 'if [ -d vendor ] ; then sudo chown -R $USER:$USER vendor; fi && make clean'
            }
        }

        stage('Build calicoq') {
            steps {
                withCredentials([sshUserPrivateKey(credentialsId: 'marvin-tigera-ssh-key', keyFileVariable: 'SSH_KEY', passphraseVariable: '', usernameVariable: '')]) {
                    sh 'if [ -z "$SSH_AUTH_SOCK" ] ; then eval `ssh-agent -s`; ssh-add $SSH_KEY || true; fi && make bin/calicoq'
                }
            }
        }

	stage('Run Unit Tests') {
            steps {
                sh 'make ut-containerized'
            }
        }

        stage('Run calicoq FVs') {
            steps {
                sh 'make fv-containerized'
            }
        }

	stage('Run STs') {
            steps {
                sh 'make st-containerized'
            }
        }

        stage('Push image to GCR') {
            steps {
                script {
                    withCredentials([file(credentialsId: 'wavetank_service_account', variable: 'DOCKER_AUTH')]) {
                        if (env.BRANCH_NAME == 'master') {
                            sh "cp $DOCKER_AUTH key.json"
                            sh "gcloud auth activate-service-account ${env.WAVETANK_SERVICE_ACCT} --key-file key.json"
                            sh "gcloud docker --authorize-only --server gcr.io"

                            sh 'make build-image'
                            sh "docker tag tigera/calicoq:latest ${env.IMAGE_NAME}:${env.BRANCH_NAME}"
                            sh "docker push ${env.IMAGE_NAME}:${env.BRANCH_NAME}"

                            // Clean up images.
                            // Hackey since empty displayed tags are not empty according to gcloud filter criteria
                            sh """
                                for digest in \$(gcloud container images list-tags ${env.IMAGE_NAME} --format='get(digest)'); do
                                if ! test \$(echo \$(gcloud container images list-tags ${env.IMAGE_NAME} --filter=digest~\${digest}) | awk '{print \$6}'); then
                                    gcloud container images delete -q --force-delete-tags "${env.IMAGE_NAME}@\${digest}"
                                fi
                                done
                            """
                        }
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
