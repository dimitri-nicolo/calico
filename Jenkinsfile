#!groovy
pipeline{
    agent { label 'containers' }
    parameters {
        string(name: 'calicoctl_url', defaultValue: 'gs://tigera-essentials/calicoctl-v2.0.0-cnx-beta1', description: 'URL of calicoctl to use in tests')
    }
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
            }
        }

        stage('Build tigera/cnx-node') {
            steps {
                ansiColor('xterm') {
                    dir('calico_node'){
                        // clear glide cache
                        sh 'sudo rm -rf ~/.glide/*'
                        sh 'make clean'
                        sh 'if [ -z "$SSH_AUTH_SOCK" ] ; then eval `ssh-agent -s`; ssh-add || true; fi && make tigera/cnx-node && docker run --rm tigera/cnx-node:latest versions'
                    }
                }
            }
        }

        stage('Get enterprise calicoctl') {
            steps {
                dir('calico_node'){
                    // Get calicoctl
		     // TODO: Matt L: remove the url and pulling release versions when it is verified that pulling from images works correctly
                    // sh "gsutil cp ${params.calicoctl_url} ./dist/calicoctl"
                    // sh "chmod +x ./dist/calicoctl"
		     sh 'make dist/calicoctl'
                }
            }
        }

        stage('Run tigera/cnx-node FVs') {
            steps {
                ansiColor('xterm') {
                    dir('calico_node'){
                        // The following bit of nastiness works round a docker issue with ttys.
                        // See http://stackoverflow.com/questions/29380344/docker-exec-it-returns-cannot-enable-tty-mode-on-non-tty-input for more
			sh 'ssh-keygen -f "/home/jenkins/.ssh/known_hosts" -R localhost'
                        sh 'ssh -o StrictHostKeyChecking=no localhost -t -t "cd $WORKSPACE/calico_node && RELEASE_STREAM=master make st"'
                    }
                }
            }
        }

	stage('Push image to GCR') {
            steps {
                script{
		    // Will eventually want to only push for passing builds. Cannot for now since the builds don't all pass currently
                    // if (env.BRANCH_NAME == 'master' && (currentBuild.result == null || currentBuild.result == 'SUCCESS')) {
                    if (env.BRANCH_NAME == 'master') {
                        sh 'docker tag tigera/cnx-node:latest gcr.io/tigera-dev/cnx/tigera/cnx-node:master'
                        sh 'gcloud docker -- push gcr.io/tigera-dev/cnx/tigera/cnx-node:master'

			// Clean up images.
			// Hackey since empty displayed tags are not empty according to gcloud filter criteria
			sh '''for digest in $(gcloud container images list-tags gcr.io/tigera-dev/cnx/tigera/cnx-node --format='get(digest)'); do
				if ! test $(echo $(gcloud container images list-tags gcr.io/tigera-dev/cnx/tigera/cnx-node --filter=digest~${digest}) | awk '{print $6}'); then
					gcloud container images delete -q --force-delete-tags "gcr.io/tigera-dev/cnx/tigera/cnx-node@${digest}"
				fi
			done'''
                    }
                }
            }
        }
    }
  post {
    always {
      junit("**/calico_node/nosetests.xml")
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
