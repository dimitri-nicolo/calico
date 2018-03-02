pipeline{
    agent { label 'slave' }
    triggers{
        pollSCM('H/5 * * * *')
        cron('H H(0-7) * * 1-5')
    }
    environment {
        IMAGE_NAME = "gcr.io/unique-caldron-775/cnx/tigera/calicoctl"
        WAVETANK_SERVICE_ACCT = "wavetank@unique-caldron-775.iam.gserviceaccount.com"
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
                    BUILD_INFO=${env.RUN_DISPLAY_URL}""".stripIndent()
                }
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
                            sh "docker tag tigera/calicoctl:latest ${env.IMAGE_NAME}:${env.BRANCH_NAME}"
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
                    slackSend message: "Failure during ${env.JOB_NAME} CI!\n${env.RUN_DISPLAY_URL}", color: "warning", channel: "cnx-ci-failures"
                }
            }
        }
    }
}
