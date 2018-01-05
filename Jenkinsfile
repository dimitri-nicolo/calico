#!groovy
pipeline {
    agent { label 'slave1' }
    triggers{
        pollSCM('H/5 * * * *')
        cron('H H(0-7) * * *')
    }
    stages {
        stage('Checkout') {
            steps {
                checkout scm
            }
        }

        stage('Install Deps') {
            steps {
                ansiColor('xterm') {
                    // Needed to allow checkout of private repos
                    sh 'if [ -z "$SSH_AUTH_SOCK" ] ; then eval `ssh-agent -s`; ssh-add || true; fi && make vendor'
                }
            }
        }
        stage('Check Typha pin') {
            steps {
                sh "echo 'Check Type pin'"
                sh "make check-typha-pins"
            }
        }
        stage('Build felix') {
            steps {
                sh "echo 'Build Felix'"
                sh "make tigera/felix"
            }
        }

        stage('Unit Tests') {
            steps {
                ansiColor('xterm') {
                    sh "echo 'Run unit Tests' && make ut-no-cover"
                }
            }
        }

        stage('Push image to GCR') {
            steps {
                script{
		    // Will eventually want to only push for passing builds. Cannot for now since the builds don't all pass currently
                    // if (env.BRANCH_NAME == 'master' && (currentBuild.result == null || currentBuild.result == 'SUCCESS')) {
                    if (env.BRANCH_NAME == 'master') {
			 sh 'make tigera/felix'
			 sh 'docker tag tigera/felix:latest gcr.io/tigera-dev/cnx/tigera/felix:master'
                        sh 'gcloud docker -- push gcr.io/tigera-dev/cnx/tigera/felix:master'

			// Clean up images.
			// Hackey since empty displayed tags are not empty according to gcloud filter criteria
			sh '''for digest in $(gcloud container images list-tags gcr.io/tigera-dev/cnx/tigera/felix --format='get(digest)'); do 
				if ! test $(echo $(gcloud container images list-tags gcr.io/tigera-dev/cnx/tigera/felix --filter=digest~${digest}) | awk '{print $6}'); then
					gcloud container images delete -q --force-delete-tags "gcr.io/tigera-dev/cnx/tigera/felix@${digest}" 
				fi 
			done'''
                    }
                }
            }
        }
    }
    post {
        always {
          junit("*/junit.xml")
          deleteDir()
        }
        success {
          echo "Yay, we passed."
        }
        failure {
          echo "Boo, we failed."
          script {
            if (env.BRANCH_NAME == 'master') {
                slackSend message: "Failure during felix-private:master CI!\nhttp://localhost:8080/view/Essentials/job/Tigera/job/felix-private/job/master/", color: "warning", channel: "cnx-ci-failures"
            }
          }
        }
    }
}
