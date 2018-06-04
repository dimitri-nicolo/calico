pipeline{
    agent { label 'slave' }
    triggers{
        pollSCM('H/5 * * * *')
        cron('H H(0-7) * * 1-5')
    }
    environment {
        NODE_IMAGE_NAME = "gcr.io/unique-caldron-775/cnx/tigera/cnx-node"

        WAVETANK_SERVICE_ACCT = "wavetank@unique-caldron-775.iam.gserviceaccount.com"

        SANE_JOB_NAME = "${env.JOB_BASE_NAME}".replace('.', '-')
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
                    CHECKOUT_BRANCH = (env.CHECKOUT_BRANCH ==~ /(master)/) ? env.CHECKOUT_BRANCH : env.CHANGE_BRANCH
                    CLONE_ORG = env.CHANGE_FORK ? env.CHANGE_FORK : 'tigera'
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
        success {
            echo "Yay, we passed."
        }
        changed { // Notify only on change to success
            script {
                if (env.BRANCH_NAME ==~ /(master)/) {
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
                if (env.BRANCH_NAME ==~ /(master)/) {
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
