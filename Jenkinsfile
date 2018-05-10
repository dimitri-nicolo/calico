pipeline{
    agent { label 'slave' }
    triggers{
        pollSCM('H/5 * * * *')
        cron('H H(0-7) * * 1-5')
    }
    environment {
        GIT_DOCS_ONLY = ""
        NODE_IMAGE_NAME = "gcr.io/unique-caldron-775/cnx/tigera/cnx-node"

        WAVETANK_SERVICE_ACCT = "wavetank@unique-caldron-775.iam.gserviceaccount.com"

        SANE_JOB_NAME = "${env.JOB_BASE_NAME}".replace('.', '-')
        BUILD_INSTANCE_NAME = "wt-${SANE_JOB_NAME}-${env.BUILD_NUMBER}".toLowerCase()
        CHECKOUT_BRANCH = "${env.GIT_BRANCH}"
    }
    stages {
        stage('Checkout') {
            steps {
                checkout scm
                script {
                    currentBuild.description = """
                    BRANCH_NAME=${env.BRANCH_NAME}
                    JOB_NAME=${env.JOB_NAME}
                    NODE_IMAGE_NAME=${env.NODE_IMAGE_NAME}:${env.BRANCH_NAME}
                    BUILD_INFO=${env.RUN_DISPLAY_URL}""".stripIndent()
                }
		script {
                    CHECKOUT_BRANCH = (env.CHECKOUT_BRANCH ==~ /(master|release-v.*)/) ? env.CHECKOUT_BRANCH : env.CHANGE_BRANCH
                    CLONE_ORG = env.CHANGE_FORK ? env.CHANGE_FORK : 'tigera'
                }
            }
        }
        stage('Check docs-only') {
            steps {
                dir('calico') {
                    script {
                        GIT_DOCS_ONLY = sh(returnStatus: true, script: "git diff --name-only HEAD^ | grep '^calico_node/' || git diff --name-only HEAD^ | grep '^_data/versions.yml'")
                    }
                }
            }
        }

        stage('Skip test evaluation') {
            when {
                expression { GIT_DOCS_ONLY == 1 }
            }
            steps {
                echo "[INFO] Only doc changes found, will skip tests"
            }
        }

        stage('Prep GCE Build instance') {
            when {
                expression { GIT_DOCS_ONLY == 0 }
                // only run if nothing is returned from running git diff grep queries
            }
            steps {
                withCredentials([sshUserPrivateKey(credentialsId: 'marvin-tigera-ssh-key', keyFileVariable: 'SSH_KEY', passphraseVariable: '', usernameVariable: '')]) {
                    sh '''
                        gcloud config set compute/zone us-central1-f && \
                        gcloud config set project unique-caldron-775 && \
                        gcloud compute instances create \
                        --machine-type n1-standard-2 \
                        --boot-disk-size 200GB \
                        --image-project ubuntu-os-cloud \
                        --image-family ubuntu-1604-lts \
                        $BUILD_INSTANCE_NAME && \
                        sleep 30
                    '''
                    sh 'gcloud compute scp $SSH_KEY ubuntu@${BUILD_INSTANCE_NAME}:.ssh/id_rsa'
                    sh '''
                        gcloud compute ssh ubuntu@${BUILD_INSTANCE_NAME} -- 'export CLOUD_SDK_REPO="cloud-sdk-$(lsb_release -c -s)" && \
                        echo "deb http://packages.cloud.google.com/apt $CLOUD_SDK_REPO main" | sudo tee -a /etc/apt/sources.list.d/google-cloud-sdk.list && \
                        curl https://packages.cloud.google.com/apt/doc/apt-key.gpg | sudo apt-key add - && \
                        sudo apt-get update && \
                        sudo apt-get install -y docker.io make google-cloud-sdk && \
                        sudo usermod -aG docker ubuntu && \
                        ssh-keyscan -t rsa github.com 2>&1 >> .ssh/known_hosts && \
                        chmod 600 .ssh/id_rsa'
                    '''
                }
            }
        }

        stage('Build tigera/cnx-node') {
            when {
                expression { GIT_DOCS_ONLY == 0 }
                // only run if nothing is returned from running git diff grep queries
            }
            steps {
                sh """
                    gcloud compute ssh ubuntu@${BUILD_INSTANCE_NAME} -- 'eval `ssh-agent -s`; ssh-add .ssh/id_rsa && \
                    git clone -b $CHECKOUT_BRANCH git@github.com:$CLONE_ORG/calico-private.git && \
                    cd calico-private/calico_node && \
                    FELIX_VER=master make tigera/cnx-node && \
                    docker run --rm tigera/cnx-node:latest versions'
                """
            }
        }

        stage('Push image to GCR') {
            when {
                expression { GIT_DOCS_ONLY == 0 }
                // only run if nothing is returned from running git diff grep queries
            }
            steps {
                script{
                    // Will eventually want to only push for passing builds. Cannot for now since the builds don't all pass currently
                    // Do that by moving this block to AFTER the tests.
                    withCredentials([file(credentialsId: 'wavetank_service_account', variable: 'DOCKER_AUTH')]) {
                        if (env.BRANCH_NAME ==~ /(master|release-.*)/) {
                            sh 'gcloud compute scp $DOCKER_AUTH ubuntu@${BUILD_INSTANCE_NAME}:key.json'
                            sh "gcloud compute ssh ubuntu@${BUILD_INSTANCE_NAME} -- 'gcloud auth activate-service-account ${WAVETANK_SERVICE_ACCT} --key-file key.json'"
                            sh "gcloud compute ssh ubuntu@${BUILD_INSTANCE_NAME} -- 'gcloud auth configure-docker'"
                            sh "gcloud compute ssh ubuntu@${BUILD_INSTANCE_NAME} -- 'docker tag tigera/cnx-node:latest ${NODE_IMAGE_NAME}:${env.BRANCH_NAME}'"
                            sh "gcloud compute ssh ubuntu@${BUILD_INSTANCE_NAME} -- 'docker push ${NODE_IMAGE_NAME}:${env.BRANCH_NAME}'"

                            // Clean up images.
                            // Hackey since empty displayed tags are not empty according to gcloud filter criteria
                            sh """
                                for digest in \$(gcloud container images list-tags ${env.NODE_IMAGE_NAME} --format='get(digest)'); do
                                if ! test \$(echo \$(gcloud container images list-tags ${env.NODE_IMAGE_NAME} --filter=digest~\${digest}) | awk '{print \$6}'); then
                                    gcloud container images delete -q --force-delete-tags "${env.NODE_IMAGE_NAME}@\${digest}" || true
                                fi
                                done
                            """
                        }
                    }
                }
            }
        }

        stage('Get enterprise calicoctl') {
            when {
                expression { GIT_DOCS_ONLY == 0 }
                // only run if nothing is returned from running git diff grep queries
            }
            steps {
                sh "gcloud compute ssh ubuntu@${BUILD_INSTANCE_NAME} -- 'cd calico-private/calico_node && make dist/calicoctl'"
            }
        }

        stage('Run tigera/cnx-node FVs') {
            when {
                expression { GIT_DOCS_ONLY == 0 }
                // only run if nothing is returned from running git diff grep queries
            }
            steps {
               sh "gcloud compute ssh ubuntu@${BUILD_INSTANCE_NAME} -- 'cd calico-private/calico_node && RELEASE_STREAM=master make st'"
            }
            post {
                always {
                    sh "gcloud compute scp ubuntu@${BUILD_INSTANCE_NAME}:calico-private/calico_node/nosetests.xml nosetests.xml"
                    junit("nosetests.xml")
                }
            }
        }

        stage('Run htmlproofer') {
            steps {
                sh 'JEKYLL_UID=10000 make htmlproofer'
            }
        }

        stage('Build docs') {
            steps {
                script {
                    if (env.BRANCH_NAME == 'master') {
                        sh 'JEKYLL_UID=10000 make publish-cnx-docs 2>&1'
                    }
                }
            }
        }

        stage('Update docs.tigera.io') {
            steps {
                script {
                    withCredentials([file(credentialsId: 'wavetank_service_account', variable: 'DOCKER_AUTH')]) {
                        if (env.BRANCH_NAME == 'master') {
                            sh "gcloud auth activate-service-account ${WAVETANK_SERVICE_ACCT} --key-file $DOCKER_AUTH --project=tigera-docs"
                            sh "gcloud app deploy --project=tigera-docs publish-cnx-docs.yaml --stop-previous-version --promote"
                        }
                    }
                }
            }
        }
    }
    post {
        always {
            script {
                if( GIT_DOCS_ONLY == 0 ){
                    // only run if nothing is returned from running git diff grep queries
                    // which determined if the gce build instance was launched
                    sh "gcloud compute instances delete --quiet ${env.BUILD_INSTANCE_NAME} --project unique-caldron-775 --zone us-central1-f"
                }
            }
        }
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
