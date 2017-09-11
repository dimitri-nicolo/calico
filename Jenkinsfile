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

	stage('Clean artifacts') {
            steps {
                sh 'if [ -d vendor ] ; then sudo chown -R $USER:$USER vendor; fi && make clean'
            }
        }

        stage('Build calicoq') {
            steps {
                sh 'if [ -z "$SSH_AUTH_SOCK" ] ; then eval `ssh-agent -s`; ssh-add || true; fi && make bin/calicoq'
            }
        }

	stage('Run Unit Tests') {
            steps {
                ansiColor('xterm') {
                    sh 'if [ -z "$SSH_AUTH_SOCK" ] ; then eval `ssh-agent -s`; ssh-add || true; fi && make ut-containerized'
                }
            }
        }

        stage('Run calicoq FVs') {
            steps {
                ansiColor('xterm') {
                    sh 'if [ -z "$SSH_AUTH_SOCK" ] ; then eval `ssh-agent -s`; ssh-add || true; fi && make fv-containerized'
                }
            }
        }

	stage('Run STs') {
            steps {
                ansiColor('xterm') {
                    sh 'if [ -z "$SSH_AUTH_SOCK" ] ; then eval `ssh-agent -s`; ssh-add || true; fi && make st-containerized'
                }
            }
        }

        stage('Push image to GCR') {
            steps {
                script{
		    // Will eventually want to only push for passing builds. Cannot for now since the builds don't all pass currently
                    // if (env.BRANCH_NAME == 'master' && (currentBuild.result == null || currentBuild.result == 'SUCCESS')) {
                    if (env.BRANCH_NAME == 'master') {
			sh 'make build-image'
			sh 'docker tag calico/calicoq:latest gcr.io/tigera-dev/calico/calicoq:latest'
                        sh 'gcloud docker -- push gcr.io/tigera-dev/calico/calicoq'
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
    }
  }
}
