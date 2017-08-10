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

        stage('Run Unit Tests') {
            steps {
                ansiColor('xterm') {
                    sh 'if [ -z "$SSH_AUTH_SOCK" ] ; then eval `ssh-agent -s`; ssh-add || true; fi && make test-containerized'
                    sh "make node-test-containerized"
                }
            }
        }

        stage('Build calico/node') {
            steps {
                sh "make calico/node"
            }
        }

        stage('Build calico/ctl') {
            steps {
                sh "make calico/ctl"
            }
        }

        stage('Build cross platform calicoctl') {
            steps {
                sh "make dist/calicoctl-darwin-amd64 dist/calicoctl-windows-amd64.exe"
            }
        }

        stage('Run calicoctl FVs') {
            steps {
                ansiColor('xterm') {
                    // The following bit of nastiness works round a docker issue with ttys.
                    // See http://stackoverflow.com/questions/29380344/docker-exec-it-returns-cannot-enable-tty-mode-on-non-tty-input for more
                    sh 'ssh localhost -t -t "cd $WORKSPACE && make st"'
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
