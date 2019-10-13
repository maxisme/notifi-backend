void setBuildStatus(String message, String state) {
  step([
      $class: "GitHubCommitStatusSetter",
      reposSource: [$class: "ManuallyEnteredRepositorySource", url: "https://github.com/maxisme/notifi-backend"],
      contextSource: [$class: "ManuallyEnteredCommitContextSource", context: "ci/jenkins/build-status"],
      errorHandlers: [[$class: "ChangingBuildStatusErrorHandler", result: "UNSTABLE"]],
      statusResultSource: [ $class: "ConditionalStatusResultSource", results: [[$class: "AnyBuildResult", message: message, state: state]] ]
  ]);
}

node() {
    try{
        checkout scm
        docker.image('mysql:5').withRun('-e "MYSQL_ROOT_PASSWORD=root"') { c ->
            def goImage = docker.build("notifi:latest", ".")
            goImage.inside("--link ${c.id}:db") {
                stage('Test'){
                    sh 'cd $WORKSPACE && encryption_key=UH9ax500yN4mnTO60WLY2ae943tsqzFw test_db_host="root:root@tcp(db:3306)" db="root:root@tcp(db:3306)/notifi_test" go test'
                }
            }
        }
        stage('Deploy'){
            sh 'ssh -o StrictHostKeyChecking=no jenk@notifi.it "sudo /bin/bash /root/notifi-backend/deploy.sh"'
        }
        setBuildStatus("Build succeeded", "SUCCESS");
    } catch (err) {
        setBuildStatus("Build failed", "FAILURE");
    }

    deleteDir()
}