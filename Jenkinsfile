pipeline{
    agent { label 'slave' }
    triggers{
        pollSCM('H/5 * * * *')
        cron('H H(0-7) * * *')
    }
    environment {
        BRANCH_NAME = 'master'
        IMAGE_NAME = "gcr.io/unique-caldron-775/cnx/tigera/calicoctl"
        IMAGE_TAG = "${env.BRANCH_NAME}"
        WAVETANK_SERVICE_ACCT = "wavetank@unique-caldron-775.iam.gserviceaccount.com"
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
                    IMAGE_NAME=${env.IMAGE_NAME}:${env.IMAGE_TAG}
                    BUILD_INFO=${env.BUILD_INFO}""".stripIndent()
                }
                git(url: 'git@github.com:tigera/calicoctl-private.git', credentialsId: 'marvin-tigera-ssh-key', branch: "${env.BRANCH_NAME}")
            }
        }
        stage('Build calicoctl') {
            steps {
                withCredentials([sshUserPrivateKey(credentialsId: 'marvin-tigera-ssh-key', keyFileVariable: 'SSH_KEY', passphraseVariable: '', usernameVariable: '')]) {
                    // Make sure no one's left root owned files in glide cache
                    // sh 'sudo chown -R ${USER}:${USER} ~/.glide'
                    // SSH_AUTH_SOCK stuff needed to allow jenkins to download from private repo
                    sh 'if [ -z "$SSH_AUTH_SOCK" ] ; then eval `ssh-agent -s`; ssh-add $SSH_KEY || true; fi && make dist/calicoctl'
                }
            }
        }
        stage('Run calicoctl FVs') {
            steps {
                sh "make st"
            }
        }
        stage('Push image to GCR') {
            steps {
                script{
                    withCredentials([file(credentialsId: 'wavetank_service_account', variable: 'DOCKER_AUTH')]) {
                        // Will eventually want to only push for passing builds. Cannot for now since the builds don't all pass currently
                        // if (env.BRANCH_NAME == 'master' && (currentBuild.result == null || currentBuild.result == 'SUCCESS')) {
                        if (env.BRANCH_NAME == 'master') {
                            sh 'make tigera/calicoctl'

                            sh "cp $DOCKER_AUTH key.json"
                            sh "gcloud auth activate-service-account ${env.WAVETANK_SERVICE_ACCT} --key-file key.json"
                            sh "gcloud docker --authorize-only --server gcr.io"
                            sh "docker tag tigera/calicoctl:latest ${env.IMAGE_NAME}:${env.IMAGE_TAG}"
                            sh "docker push ${env.IMAGE_NAME}:${env.IMAGE_TAG}"

                            // Clean up images.
                            // Hackey since empty displayed tags are not empty according to gcloud filter criteria
                            sh '''
                                for digest in $(gcloud container images list-tags ${env.IMAGE_NAME} --format='get(digest)'); do
                                    if ! test $(echo $(gcloud container images list-tags ${env.IMAGE_NAME} --filter=digest~${digest}) | awk '{print $6}'); then
                                    gcloud container images delete -q --force-delete-tags "${env.IMAGE_NAME}@${digest}"
                                    fi
                                done
                            '''
                        }
                    }
                }
            }
        }
    }
    post {
        always {
            junit("nosetests.xml")
            deleteDir()
        }
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
