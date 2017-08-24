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

	/* Comment this out until the unit tests are merged
        stage('Run Unit Tests') {
            steps {
                ansiColor('xterm') {
                    sh 'if [ -z "$SSH_AUTH_SOCK" ] ; then eval `ssh-agent -s`; ssh-add || true; fi && make ut-containerized'
                }
            }
        }
	*/

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
