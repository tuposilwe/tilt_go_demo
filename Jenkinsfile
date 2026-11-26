pipeline {
    agent any

    options {
        timestamps()
        disableConcurrentBuilds()
        buildDiscarder(logRotator(numToKeepStr: '10'))
        timeout(time: 30, unit: 'MINUTES')
    }

    environment {
        IMAGE_NAME     = 'tuposilwe/tilt_go_demo'
        HELM_RELEASE   = 'tilt-go-demo'
        KUBE_NAMESPACE = 'tilt-go-demo'
    }

    triggers {
        githubPush()
    }

    stages {
        stage('Deploy-master pipeline') {
            when { branch 'deploy-master' }

            stages {
                stage('Checkout') {
                    steps {
                        checkout scmGit(
                            branches: [[name: '*/deploy-master']],
                            userRemoteConfigs: scm.userRemoteConfigs
                        )
                        script {
                            env.IMAGE_VERSION = fileExists('VERSION') ? readFile('VERSION').trim() : '0.1.0'
                            env.GIT_SHA       = sh(
                                script: 'git rev-parse --short=12 HEAD',
                                returnStdout: true
                            ).trim()
                            env.IMAGE_TAG     = "${env.IMAGE_VERSION}-${env.GIT_SHA}"
                        }
                    }
                }

                stage('Test Go') {
                    steps {
                        sh '''
                            go vet ./...
                            go test -race -v ./...
                        '''
                    }
                }

                stage('Docker Build') {
                    steps {
                        sh """
                            docker build -f Dockerfile \
                              -t ${IMAGE_NAME}:${IMAGE_TAG} \
                              -t ${IMAGE_NAME}:latest .
                        """
                    }
                }

                stage('Docker Push') {
                    steps {
                        withCredentials([usernamePassword(
                                credentialsId: 'dockerhub-creds',
                                usernameVariable: 'DH_USER',
                                passwordVariable: 'DH_PASS')]) {
                            sh '''
                                echo "$DH_PASS" | docker login -u "$DH_USER" --password-stdin
                                docker push ${IMAGE_NAME}:${IMAGE_TAG}
                                docker push ${IMAGE_NAME}:latest
                                docker logout
                            '''
                        }
                    }
                }

                stage('Deploy') {
                    steps {
                        sshagent(credentials: ['dev-server-ssh']) {
                            dir('ansible') {
                                sh """
                                    ansible-playbook deploy.yml \
                                      -e image_name=${IMAGE_NAME} \
                                      -e image_tag=${IMAGE_TAG} \
                                      -e release_name=${HELM_RELEASE} \
                                      -e namespace=${KUBE_NAMESPACE}
                                """
                            }
                        }
                    }
                }
            }

            post {
                success {
                    echo "Deployed ${IMAGE_NAME}:${IMAGE_TAG}"
                }
                failure {
                    echo 'Pipeline failed. Check the stage logs above.'
                }
                cleanup {
                    script {
                        if (env.IMAGE_TAG) {
                            sh "docker rmi ${IMAGE_NAME}:${IMAGE_TAG} ${IMAGE_NAME}:latest || true"
                        }
                    }
                    cleanWs()
                }
            }
        }
    }
}