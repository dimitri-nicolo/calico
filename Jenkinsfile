#!groovy
pipeline{
    agent { label 'containers' }
    triggers{
        pollSCM('H/5 * * * *')
        cron('H H(0-7) * * *')
    }
    stages {
        stage('Check clean slate') {
	    // It is a critical problem if we are not starting from a
	    // clean position; so we intentionally fail the entire
	    // pipeline if any of the following checks fail.
	    steps {
	        // Check nothing listening on the etcd ports.
	        sh "! sudo ss -tnlp 'sport = 2379' | grep 2379"
	        sh "! sudo ss -tnlp 'sport = 2380' | grep 2380"
	    }
        }

        stage('Checkout') {
            steps {
                checkout scm
            }
        }

        stage('Wipe out all docker state') {
            steps {
                // Kill running containers:
                sh "sudo docker kill `docker ps -qa` || true"
                // Delete all containers (and their associated volumes):
                sh "sudo docker rm -v -f `docker ps -qa` || true"
                // Remove all images:
                sh "sudo docker rmi -f `docker images -q` || true"
                // clear glide cache
                sh 'sudo rm -rf ~/.glide/*'
                }
        }

        stage('Build calicoctl') {
            steps {
                // Make sure no one's left root owned files in glide cache
                sh 'sudo chown -R ${USER}:${USER} ~/.glide'
                // SSH_AUTH_SOCK stuff needed to allow jenkins to download from private repo
                sh 'if [ -z "$SSH_AUTH_SOCK" ] ; then eval `ssh-agent -s`; ssh-add || true; fi && make dist/calicoctl'
            }
        }

        stage('Run calicoctl FVs') {
            steps {
                ansiColor('xterm') {
                    sh "make st"
                }
            }
        }

        stage('Push image to GCR') {
            steps {
                script{
                    // Will eventually want to only push for passing builds. Cannot for now since the builds don't all pass currently
                    // if (env.BRANCH_NAME == 'master' && (currentBuild.result == null || currentBuild.result == 'SUCCESS')) {
                    if (env.BRANCH_NAME == 'master') {
                    sh 'make tigera/calicoctl'
                    sh 'docker tag tigera/calicoctl:latest gcr.io/tigera-dev/cnx/tigera/calicoctl:master'
                    sh 'gcloud docker -- push gcr.io/tigera-dev/cnx/tigera/calicoctl:master'

                    // Clean up images.
                    // Hackey since empty displayed tags are not empty according to gcloud filter criteria
                    sh '''for digest in $(gcloud container images list-tags gcr.io/tigera-dev/cnx/tigera/calicoctl --format='get(digest)'); do
                            if ! test $(echo $(gcloud container images list-tags gcr.io/tigera-dev/cnx/tigera/calicoctl --filter=digest~${digest}) | awk '{print $6}'); then
                              gcloud container images delete -q --force-delete-tags "gcr.io/tigera-dev/cnx/tigera/calicoctl@${digest}"
                            fi
                          done'''
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
                slackSend message: "Failure during calicoctl-private:master CI!\nhttp://localhost:8080/view/Essentials/job/Tigera/job/calicoctl-private/job/master/", color: "warning", channel: "cnx-ci-failures"
            }
        }
    }
  }
}
