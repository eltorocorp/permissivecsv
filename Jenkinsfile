def setCoverageBadge(goPath) {
    def coverage = sh(script: "cd ${goPath} && cat coveragepct", returnStdout: true)
    def coverageUri = "\'http://badges.awsp.eltoro.com?project=permissivecsv&item=coverage&value=${coverage}&color=yellow\'"
    sh "curl -sX POST ${coverageUri}"
}

def setBuildStatusBadge(status, color) {
    def statusUri = "\'http://badges.awsp.eltoro.com?project=permissivecsv&item=build&value=${status}&color=${color}\'"
    sh "curl -sX POST ${statusUri}"
}

def slackSuccess() {
    def slack_message = "permissivecsv build succeeded!"
    slackSend channel: '#dev-badass-badgers', message: "${slack_message}", failOnError:true, tokenCredentialId: 'slack-token', color:"good"
}

def slackFailure(){
    def slack_message = "permissivecsv build failed! Details: ${BUILD_URL}"
    slackSend channel: '#dev-badass-badgers', message: "${slack_message}", failOnError:true, tokenCredentialId: 'slack-token', color:"danger"
}

node {
    String goPath = "/go/src/github.com/eltorocorp/permissivecsv"
    docker.image("golang:1.11").inside("-v ${pwd()}:${goPath} -u root") {
        try {
            stage('Pre-Build') {
                setBuildStatusBadge('pending', 'blue')
                sh "chmod -R 0777 ${goPath}"
                checkout scm
                sh "cd ${goPath} && make prebuild"
            }

            stage('Build') {
                sh "cd ${goPath} && make build"
            }

            stage('Test') {
                sh "cd ${goPath} && make test"
            }

            stage("Post-Build") {
                setBuildStatusBadge('passing', 'green')
                slackSuccess()
                currentBuild.result = 'SUCCESS'
            }
        } catch (Exception err) {
            sh "echo ${err}"
            slackFailure()
            setBuildStatusBadge('failing', 'red')
            currentBuild.result = 'FAILURE'
        } finally {
            setCoverageBadge(goPath)
        }  
    }
}