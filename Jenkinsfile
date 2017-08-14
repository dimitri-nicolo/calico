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

        stage('Build calicoctl') {
            steps {
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
