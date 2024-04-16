pipeline {
    agent {
        docker { image 'golang:1.22' }
    }
    stages {
        stage('Test') {
            steps {
                sh 'go version'
            }
        }
    }
}