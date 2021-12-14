pipeline {
    agent {
        label 'slave'
    }

    stages {
        stage('Build') {
            steps {
                withCredentials([sshUserPrivateKey(credentialsId: 'marvin-tigera-ssh-key', keyFileVariable: 'SSH_KEY', passphraseVariable: '', usernameVariable: '')]) {
                    // Run SSH agent with marvin's ssh key so that 'dep ensure' can install private repos (tigera/licensing, libcalico-go-private, etc)
                    sh 'eval `ssh-agent -s`; ssh-add $SSH_KEY; make build'
                }
            }
        }
        stage('Test') {
            steps {
                sh "make test"
            }
        }
    }
}
