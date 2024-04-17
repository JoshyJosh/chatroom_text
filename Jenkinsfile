pipeline {
    agent any 
    environment {
        BUILDERIMAGE = "builder-image"
    }

    stages {
        stage('Build base stage') {
            steps {
                sh 'docker build -t ${BUILDERIMAGE} --target builder .'
            }
        }

        stage('Run tests') {
            steps {
                sh 'docker run --rm ${BUILDERIMAGE} sh -c "go test ./..."'
            }
        }
    }
}