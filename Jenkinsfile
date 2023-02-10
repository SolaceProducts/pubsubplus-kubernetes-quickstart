currentBuild.description = 'Cause: version bump'
node(label: 'centos7_router_devserver') {
    
    library 'jenkins-pipeline-library@main'
    cleanWs()

    String KUBERNETES_BRANCH = env.BRANCH_NAME
    stage("kubernetes-operator-build") {
        job = build job: 'kubernetes-operator-build', propagate: true, parameters:
        [
            string(name: 'KUBERNETES_BRANCH', value: KUBERNETES_BRANCH),
        ]
    }
}

