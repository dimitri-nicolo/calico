#!groovy
properties([
    pipelineTriggers([
      [$class: "SCMTrigger", scmpoll_spec: "H/5 * * * *"],
    ])
  ])
node {
    stage('Checkout') {
        checkout scm
    }

    wrap([$class: 'AnsiColorBuildWrapper']) {
        stage('Install Deps') {
            // Needed to allow checkout of private repos
            sh 'if [ -z "$SSH_AUTH_SOCK" ] ; then eval `ssh-agent -s`; ssh-add || true; fi && make vendor'
        }
    }

    stage('Build felix') {
        sh "echo 'Build Felix'"
        sh "make calico/felix"
    }

    wrap([$class: 'AnsiColorBuildWrapper']) {
        stage('Unit Tests') {
            sh "echo 'Run unit Tests' && make ut-no-cover"
        }
    }

  }
  post {
    always {
      deleteDir()
    }
    success {
      echo "Yay, we passed."
    }
    failure {
      echo "Boo, we failed."
    }
  }
