currentBuild.description = 'Cause: version bump'
node(label: 'centos7_router_devserver') {

    library 'jenkins-pipeline-library@main'
    cleanWs()

    String JCSMP_BRANCH = env.BRANCH_NAME
    stage("jcsmp-build") {
        job = build job: 'opentelemetry-jcsmp-integration-build', propagate: true, parameters:
        [
            string(name: 'JCSMP_BRANCH', value: JCSMP_BRANCH),
        ]
    }
}
