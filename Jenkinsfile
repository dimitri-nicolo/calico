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

        stage('Build calico/node') {
            steps {
                dir('calico_node'){
                    sh 'if [ -z "$SSH_AUTH_SOCK" ] ; then eval `ssh-agent -s`; ssh-add || true; fi && make calico/node && docker run --rm calico/node:latest versions && make st'
                }
            }
        }

        stage('Get enterprise calicoctl') {
            steps {
                dir('calico_node'){
                    // Get calicoctl
                    sh "gsutil cp gs://tigera-essentials/calicoctl-v1.0.3-rc1 ./dist/calicoctl"
                    sh "chmod +x ./dist/calicoctl"
                }
            }
        }

        stage('Run calico/node FVs') {
            steps {
                ansiColor('xterm') {
                    dir('calico_node'){
                        // The following bit of nastiness works round a docker issue with ttys.
                        // See http://stackoverflow.com/questions/29380344/docker-exec-it-returns-cannot-enable-tty-mode-on-non-tty-input for more
                        sh 'ssh localhost -t -t "cd $WORKSPACE/calico_node && make st"'
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
