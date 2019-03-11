pipeline{
    agent { label 'slave' }
    triggers{
        pollSCM('H/5 * * * *')
        cron('H H(0-7) * * 1-5')
    }
    environment {
        CALICOQ_IMAGE_NAME = "gcr.io/unique-caldron-775/cnx/tigera/calicoq"
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
                        CALICOQ_IMAGE_NAME=${env.CALICOQ_IMAGE_NAME}:${env.BRANCH_NAME}
                        BUILD_INFO=${env.RUN_DISPLAY_URL}""".stripIndent()
                    }
            }
        }
        stage('Clean artifacts') {
            steps {
                sh 'if [ -d vendor ] ; then sudo chown -R $USER:$USER vendor; fi && make clean'
            }
        }

        stage('Check calicoq changes') {
            steps {
                script {
                    SKIP_CALICOQ_BUILD = sh(returnStatus: true, script: "git diff --name-only HEAD^")
                    echo "SKIP_CALICOQ_BUILD=${SKIP_CALICOQ_BUILD}"
                }
            }
        }
        stage('Build calicoq') {
            when {
                expression { SKIP_CALICOQ_BUILD == 0 }
            }
             steps {
                 withCredentials([sshUserPrivateKey(credentialsId: 'marvin-tigera-ssh-key', keyFileVariable: 'SSH_KEY', passphraseVariable: '', usernameVariable: '')]) {
                     sh 'if [ -z "$SSH_AUTH_SOCK" ] ; then eval `ssh-agent -s`; ssh-add $SSH_KEY || true; fi && make bin/calicoq'
                 }
             }
        }

        stage('Run calicoq UTs') {
            when {
                expression { SKIP_CALICOQ_BUILD == 0 }
            }
            steps {
                sh 'make ut-containerized'
            }
        }

        stage('Run calicoq FVs') {
            when {
                expression { SKIP_CALICOQ_BUILD == 0 }
            }
            steps {
                sh 'make fv-containerized'
            }
        }

        stage('Run calicoq STs') {
            when {
                expression { SKIP_CALICOQ_BUILD == 0 }
            }
            steps {
                sh 'make st-containerized'
            }
        }

	stage('Run licensing checks') {
	    steps {
                script {
                     withCredentials([string(credentialsId: 'fossa_api_key', variable: 'FOSSA_API_KEY')]) {
                         withCredentials([sshUserPrivateKey(credentialsId: 'marvin-tigera-ssh-key', keyFileVariable: 'SSH_KEY', passphraseVariable: '', usernameVariable: '')]) {
                             if (env.BRANCH_NAME ==~ /(master|release-.*)/) {
                                 sh 'if [ -z "$SSH_AUTH_SOCK" ] ; then eval `ssh-agent -s`; ssh-add $SSH_KEY || true; fi && FOSSA_API_KEY=$FOSSA_API_KEY make foss-checks'
                             }
                         }
                     }
                }
            }
        }

        stage('Push calicoq image to GCR') {
            when {
                expression { SKIP_CALICOQ_BUILD == 0 }
            }
            steps {
                script {
                    withCredentials([file(credentialsId: 'wavetank_service_account', variable: 'DOCKER_AUTH')]) {
                        if (env.BRANCH_NAME ==~ /(master|release-.*)/) {
                            sh "cp $DOCKER_AUTH key.json"
                            sh "gcloud auth activate-service-account ${env.WAVETANK_SERVICE_ACCT} --key-file key.json"
                            sh "gcloud auth configure-docker"

                            sh 'make build-image'
                            sh "docker tag tigera/calicoq:latest ${env.CALICOQ_IMAGE_NAME}:${env.BRANCH_NAME}"
                            sh "docker push ${env.CALICOQ_IMAGE_NAME}:${env.BRANCH_NAME}"

                            // Clean up images.
                            // Hackey since empty displayed tags are not empty according to gcloud filter criteria
                            sh """
                                for digest in \$(gcloud container images list-tags ${env.CALICOQ_IMAGE_NAME} --format='get(digest)'); do
                                if ! test \$(echo \$(gcloud container images list-tags ${env.CALICOQ_IMAGE_NAME} --filter=digest~\${digest}) | awk '{print \$6}'); then
                                    gcloud container images delete -q --force-delete-tags "${env.CALICOQ_IMAGE_NAME}@\${digest}" || true
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
