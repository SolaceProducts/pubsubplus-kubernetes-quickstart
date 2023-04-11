properties([
    buildDiscarder(logRotator(daysToKeepStr: '365', numToKeepStr: '350')),
    parameters([
        string(name: 'GIT_SHA', description: 'Git SHA to checkout.'),
        string(name: 'KUBERNETES_BRANCH', description: 'Branch to checkout. <b>Will not use if GIT_SHA is set.</b>'),
        booleanParam(name: 'FINAL_CUT', defaultValue: false, description: 'Automation parameter, do not touch')
    ])
])

library 'jenkins-pipeline-library@main'

boolean FINAL_CUT = params.FINAL_CUT
String gitSha = params.GIT_SHA
String gitShaShort = ""
String kubernetesBranch = params.KUBERNETES_BRANCH
String internalRegistry = 'apps-jenkins:18888/kubernetes-operator'
String version = ""

if (gitSha == "") {
    if (kubernetesBranch == "") {
        error("This build requires either a git sha or a branch. Please check upstream job!")
    }
    println "Git SHA not provided. Using branch ${kubernetesBranch} as param for checkout."
}

node(label: "centos7_fast_devserver") {
// node(label: "dev3-177") {
    notify(slackChannel: '#api-team-build-notification') {

        cleanWs()

        // Set build description
        if (gitSha == "") {
            currentBuild.description = "Source Path: pubsubplus-kubernetes-operator.git/${kubernetesBranch}"
        } else {
            currentBuild.description = "Source Path: pubsubplus-kubernetes-operator.git@${gitSha}"
        }

        // Checkout source code using either <GIT_SHA> or <KUBERNETES_BRANCH>
        stage("Checkout") {

            String branchName = gitSha
            if (branchName == "") {
                branchName = "*/${kubernetesBranch}"
            }

            println "Checkout out from branch ${branchName}"
            try {
                def checkoutResults = checkout([$class: 'GitSCM',
                    branches: [[name: branchName]],
                    doGenerateSubmoduleConfigurations: false,
                    extensions: [],
                    submoduleCfg: [],
                    userRemoteConfigs: [[credentialsId: 're-github-bot-1', url: 'git@github.com:SolaceDev/pubsubplus-kubernetes-operator.git']]
                ])

                if (gitSha == "") {
                    gitSha = checkoutResults.GIT_COMMIT
                }
                gitShaShort = gitSha.substring(0,5)
            } catch (e) {
                println "Error while checking out pubsubplus-kubernetes-operator repository."
                error(e)
            }
        }

        // Replace -SNAPSHOT depending on the following:
        //      Feature branch     --> version-SNAPSHOT
        //      Main branch        --> version-${GIT_SHA}
        //      Release branch     --> version-RC
        //      Release branch+FC  --> version
        //
        //      Release pattern currently matches: 1.0.0
        String releasePattern = /[0-9].[0-9].[0-9]$/
        boolean isReleaseBranch = false
    
        stage("Version and Save Docker Image") {
            //get version path from version.go: version ex:1.0.0
            sh "cd /opt/cvsdirs/loadbuild/jenkins/slave/workspace/kubernetes-operator-build"
            version = sh(returnStdout:true, script:"cat version.go | grep \"const version\" | sed 's/const version = \"\\(.*\\)\"/\\1/'").trim()

            String imageTag =''
            String uniqueVersion = gitShaShort
            if (kubernetesBranch == "v1.0.0"){
                if (FINAL_CUT){
                    imageTag = "1.0.0"
                } else {
                    imageTag = "1.0.0-${uniqueVersion}"
                }
            } else {
                imageTag = "${version}-${kubernetesBranch}-${uniqueVersion}"
            }
            //build docker image of pubsubplus-kubernetes-operator project
            sh "docker build -t apps-jenkins:18888/pubsubplus-eventbroker-operator:${imageTag} ."

            //save docker image as tar file
            sh "docker save apps-jenkins:18888/pubsubplus-eventbroker-operator:${imageTag} | gzip > ./pubsubplus-eventbroker-operator_${imageTag}.tar.gz"

            //make a new directory to store the tar file
            sh "mkdir -p /home/public/RND/loads/pubsubplus-eventbroker-operator/${kubernetesBranch}/${imageTag} "

            //move the tar file to the new directory
            sh "mv pubsubplus-eventbroker-operator_${imageTag}.tar.gz /home/public/RND/loads/pubsubplus-eventbroker-operator/${kubernetesBranch}/${imageTag}"
        }

        stage ('Upload image to internal registry') {
        // Login again
        withCredentials([
            string(credentialsId: 'nexus-robot2-passwd', variable: 'DOCKER_PASSWORD')]) {
            sh 'docker login apps-jenkins:18888 -u solace -p solace1'
        }
        // Copy
        sh """
            docker tag apps-jenkins:18888/pubsubplus-eventbroker-operator:${imageTag} ${internalRegistry}:${imageTag}
            docker push ${internalRegistry}:${imageTag}
            docker rmi apps-jenkins:18888/pubsubplus-eventbroker-operator:${imageTag} ${internalRegistry}:${imageTag}
        """
    }
    }
}