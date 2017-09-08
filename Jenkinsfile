#!groovy
pipeline{
    agent { label 'containers' }
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

        stage('Wipe out all docker state') {
            steps {
                // Kill running containers:
                sh "sudo docker kill `docker ps -qa` || true"
                // Delete all containers (and their associated volumes):
                sh "sudo docker rm -v `docker ps -qa` || true"
                // Remove all images:
                sh "sudo docker rmi `docker images -q` || true"
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
			 sh 'make calico/ctl'
			 sh 'docker tag calico/ctl:latest gcr.io/tigera-dev/calico/ctl-essentials:latest'
                        sh 'gcloud docker -- push gcr.io/tigera-dev/calico/ctl-essentials:latest'
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
    }
  }
}
