library "internal-cicd-shared-lib@master"
library "platform-cd@1.1.2"

// check if the branch has a prefix of release/
// if so branch simple name will be release-${NUMBER} e.g release-1.23
def getBranchSimpleName(branchName) {
    if(branchName.contains('/')  && "release".equals(branchName.substring(0, BRANCH_NAME.indexOf('/')))) {
        return branchName.replace('/', '-');
    } else {
        return branchName.substring(BRANCH_NAME.lastIndexOf('/')+1);
    }
}

def checkClusterRole(def clusterRolePath) {
    echo "checking clusterrole: ${clusterRolePath}"
    def status = sh(returnStdout: true, script: "curl -X POST -s -o clusterrole-scan.json -w '%{http_code}'  ${env.CRD_SCAN_URL} --header 'Accept: */*'  --form 'file=@${clusterRolePath}'").trim()

    echo "status: ${status}"
    sh "cat clusterrole-scan.json | jq"
    if (status != "200"){
        error("CRD scan failed for '${clusterRolePath}', expected 200 response, returned: '${status}'")
    }
}

pipeline {
  agent {
    label 'kubex'
  }
  options {
    lock resource: 'kagent_shared_lock'
    timeout(time: 1, unit: 'HOURS')
  }
  environment {
    HST = sh(returnStdout: true, script: 'hostname -s').trim()
    jobName = "${JOB_NAME.replaceAll(/\/\w+%2F/,'/').toLowerCase()}"
    BRANCH_SIMPLE_NAME = getBranchSimpleName(env.BRANCH_NAME)
    BRANCH_SIMPLE_BUILD_NAME = "${BRANCH_SIMPLE_NAME}-${BUILD_NUMBER}"
    SEMVER = "0.3.12-rc${BUILD_NUMBER}"
    DROP_VERSION = "25.06.00.00"
    TESTE2E = "OCP"
    HOME_PATH = "$HOME/workspace/${jobName}/src/bitbucket.corp.amdocs.com/plto/platform-kagent"
    KUBECONFIG = "${HOME_PATH}/.kube/config"
    RELEASE_DATE = sh(returnStdout: true, script: 'date +%d/%m/%Y').trim()
    RELEASE_NUM = ""
    BASE_GO_VERSION = "1.24.3"
    sbom_report = "SPDX:TAGVALUE"
    YARN_CACHE_FOLDER = "$HOME/workspace/${jobName}/yarn-cache"
  }

  stages {
    stage('Clean') {
        steps {
            sh "sudo rm -rf $HOME/workspace/$jobName/"
            echo 'Clean done'
             script {
                manager.addShortText("NODE: ${HST}", "black", "white", "1px", "green");
                manager.addShortText("TAG: ${BRANCH_SIMPLE_BUILD_NAME}", "black", "white", "1px", "green");
            }
        }
    }

    stage ('Semver') {
        steps {
            script {
                echo ("Selecting Semver from tag/branch ${env.BRANCH_SIMPLE_NAME}")
                echo ("HOME_PATH: $HOME_PATH")
                echo ("WORKSPACE: $WORKSPACE")
                // https://semver.org/ Semver naming convention
                // https://regex101.com/r/vkijKf/1
                def matches = ("${env.BRANCH_SIMPLE_NAME}" =~ /(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-((?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?(?:\+([0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*))?$/)
                if(matches){
                    echo ("Using extracted semver version: " + matches[0][0])
                    SEMVER = matches[0][0]
                    if (matches[0][3].length() == 1){
                        DROP_VERSION = env.DROP_VERSION + ".0" + matches[0][3]
                    }
                    else {
                        DROP_VERSION = env.DROP_VERSION + "." + matches[0][3]
                    }

                    echo ("Using extracted Drop version: " + DROP_VERSION)
                }

                echo ("Using Semver: ${SEMVER}")
                manager.addShortText("SEMVER: ${SEMVER}", "black", "white", "1px", "green");
                manager.addShortText("GO: ${BASE_GO_VERSION}", "black", "white", "1px", "green");
                sh """sleep 5""" //allow stop the pipeline to see the semver
            }
        }
    }
/*
    stage('Set Release Number') {
        when {
            // upload artifact descriptor for releases
            expression { env.SEMVER != SEMVER }
        }
        steps {
            ws( "$HOME_PATH" ){
            script {
                    def releaseNumber = input(
                            id: 'userInput', message: 'Enter Release Number:?',
                            parameters: [

                                    string(defaultValue: '',
                                            description: 'Release Number',
                                            name: 'ReleaseNumber'),
                            ])

                   RELEASE_NUM = "${releaseNumber}"
                }
            }
        }
    }
*/

    stage('Git Checkout') { // for display purposes
        steps {
            ws( "${HOME_PATH}" ) {
                echo sh(returnStdout: true, script: 'env')
                sh "export GOPATH=$HOME/workspace/${jobName};"
                sh 'echo $GOPATH;'
                checkout scm: [$class: 'GitSCM', userRemoteConfigs: [[url: 'ssh://git@bitbucket.corp.amdocs.com:7999/plto/platform-kagent.git',
                    credentialsId: 'PLTO_Jenkins_ssh']], branches: [[name: "${env.BRANCH_NAME}"]]],poll: false
                load "cfg/pipeline.groovy"
            }

            withCredentials([
                  usernamePassword(
                    credentialsId: "NEXUS_CRED_INTERNAL",
                    usernameVariable: 'NEXUS_CRED_USER',
                    passwordVariable: 'NEXUS_CRED_PASS'
                  )
            ]) {
                sh """
                  cd $HOME_PATH;
                  source $HOME/.bash_profile
                  source hack/git/.profile
                  export GOPATH=$HOME/workspace/${jobName}
                  docker login ${env.PLATFORM_DOCKER_REPO}   -u '$NEXUS_CRED_USER' -p '$NEXUS_CRED_PASS';
                  make TAG=${env.BRANCH_SIMPLE_NAME} GO_VERSION=${env.BASE_GO_VERSION} checkout/tags
                """
            }
            echo 'Git checkout done'
        }
    }

     stage('Get Latest Commit') {
            steps {
                script {
                    env.GIT_COMMIT_HASH = sh(script: 'git rev-parse HEAD', returnStdout: true).trim()
                    echo "Latest Git Commit Hash: ${env.GIT_COMMIT_HASH}"
                }
            }
        }

    stage('Build') {
        steps {
            parallel(
              kagent: {
                echo 'Docker Building..'
                sh """
                cd $HOME_PATH;
                source $HOME/.bash_profile
                export GOPATH=$HOME/workspace/${jobName}
                make SEMVER=${SEMVER} GO_VERSION=${env.BASE_GO_VERSION} HUBS=${env.PLATFORM_DOCKER_REPO}/platform/kagent builds
                """
              }
           )
        }
    }

    //Trigger blackduck scan for every build
    stage('Trigger Blackduck Scan Remotely') {
        steps {
            withCredentials([
                usernamePassword(
                    credentialsId: "KUBEX_FOSS_PASS",
                    usernameVariable: 'KUBEX_FOSS_USER',
                    passwordVariable: 'KUBEX_FOSS_PASS' )
            ]) {
            script {

            //clean go cache
            sh """
            cd $HOME_PATH;
            source $HOME/.bash_profile
            source hack/git/.profile
            export GOPATH=$HOME/workspace/${jobName}
            make TAG=${env.BRANCH_SIMPLE_NAME} GO_VERSION=${env.BASE_GO_VERSION} clean/go clean/yarn
            """

            def remoteCommand = "set +x; curl --noproxy '*' -vv -k -X POST '${env.FOSS_JENKINS}/${env.JENKINS_JOB_NAME}/buildWithParameters?token=fossscan' --user '$KUBEX_FOSS_USER:$KUBEX_FOSS_PASS' "
            remote_host = sh(script: 'hostname', returnStdout: true).trim()
            echo "Hostname: ${remote_host}"

            def productVersion = (SEMVER == "0.0.0") ? "kagent-${BRANCH_SIMPLE_NAME}" : "kagent-${SEMVER}"
            def parameters = [
                "PRODUCT_NAME=PLATFORM_KUBEX",
                "PRODUCT_VERSION=${productVersion}",
                "VERSION_TYPE=release",
                "SCAN_TYPE=ALL",
                "BLACKDUCK_USER_TOKEN=${env.BLACKDUCK_TOKEN}",
                "SKIP_BLACKDUCK_SCAN=false",
                "REMOTE_HOST=${remote_host}",
                "USERNAME=${env.KUBEX_NODE_USER}",
                "PASSWORD=${env.KUBEX_NODE_PASS}",
                "SCAN_FOLDER=$HOME/workspace/${jobName}",
                "EMAIL=${env.KUBEX_EMAIL}",
                "OPEN_JIRA_DEFECT=false",
                "EXCLUDE_DEFECTS_MISSING_SOURCES=true",
                "JIRA_URL=",
                "JIRA_PROJECT_KEY=",
                "JIRA_USERNAME=",
                "JIRA_PASSWORD=",
                "DOCKER_IMAGE_NAME=",
                "MAVEN_COMMAND=",
                "EXCLUDE_DETECTOR_PACKAGE=",
                "ADDITIONAL_DETECT_PARAMETERS=",
                "JIRA_REOPEN_FROM=",
                "OPEN_OCTANE_DEFECT=false",
                "OCTANE_URL=",
                "OCTANE_SHAREDSPACE_ID=",
                "OCTANE_WORKSPACE_ID=",
                "OCTANE_USERNAME=",
                "OCTANE_PASSWORD=",
                ]

                // Add sbom_report parameter only if it has a valid value
                if (sbom_report && sbom_report != "NONE") {
                parameters.add("GENERATE_SBOM_REPORT=${sbom_report}")
                }

                // Add scm_and_artifact_details parameter only if the file exists
                def scmFilePath = "${WORKSPACE}/scm_and_artifact_details.yaml"
                if (fileExists(scmFilePath)) {
                    parameters.add("SCM_AND_ARTIFACT_DETAILS=@${scmFilePath}")
                }

                parameters.each { param -> remoteCommand += " --form '${param}' " }

                echo "Remote Command: ${remoteCommand}"

                exit_code = sh(script: "${remoteCommand}", returnStatus: true)
            }
            }
        }
    }

    stage('Copy DMZ - ARTIFACTORY') {
        steps {
            withCredentials([
              usernamePassword(
                credentialsId: "NEXUS_CRED_INTERNAL",
                usernameVariable: 'NEXUS_CRED_USER',
                passwordVariable: 'NEXUS_CRED_PASS'
              ),
              usernamePassword(
                  credentialsId: "artifactory_ci_credentials",
                  usernameVariable: 'ARTIF_CRED_USER',
                  passwordVariable: 'ARTIF_CRED_PASS'
              )
            ]) {
            sh """
            cd $HOME_PATH;
            source $HOME/.bash_profile
            source hack/git/.profile
            export GOPATH=$HOME/workspace/${jobName}
            make GO_VERSION=${env.BASE_GO_VERSION} \
                 VERSION=${SEMVER} TAG=${env.BRANCH_SIMPLE_NAME} \
                 MAVEN_DD_URL=${env.MAVEN_DD_URL}  \
                 ARTIF_CRED_USER=${ARTIF_CRED_USER} ARTIF_CRED_PASS=${ARTIF_CRED_PASS} \
                 NEXUS_CRED_USER=${NEXUS_CRED_USER} NEXUS_CRED_PASS=${NEXUS_CRED_PASS} \
                 -j2 copy-dmz copy-art
            """
            }
        }
    }
    stage('CVE Report') {
        steps {
            script {
                echo 'CVE Report..'
                sh """
                cd $HOME_PATH;
                source $HOME/.bash_profile
                source hack/git/.profile
                export GOPATH=$HOME/workspace/${jobName}
                make TAG=${env.BRANCH_SIMPLE_NAME} SEMVER=${SEMVER} GO_VERSION=${env.BASE_GO_VERSION} HUBS=${env.PLATFORM_DOCKER_REPO}/platform/kagent cve/scan
                """
            }
        }
    }
    stage('test-e2e') {
        steps {
            sh """
            ## not ready yet
            ## make TAG=${env.BRANCH_SIMPLE_NAME} SEMVER=${SEMVER} GO_VERSION=${env.BASE_GO_VERSION} HUBS=${env.PLATFORM_DOCKER_REPO}/platform/kagent test/e2e
            """
        }
    }
    stage('Upload Nexus') {
        steps {
            withCredentials([
                  usernamePassword(
                    credentialsId: "NEXUS_CRED_INTERNAL",
                    usernameVariable: 'NEXUS_CRED_USER',
                    passwordVariable: 'NEXUS_CRED_PASS'
                  ),
                  usernamePassword(
                      credentialsId: "artifactory_ci_credentials",
                      usernameVariable: 'ARTIF_CRED_USER',
                      passwordVariable: 'ARTIF_CRED_PASS'
                  )
            ]) {
            sh """
            echo "INFO: Building charts ${SEMVER}"
            cd $HOME_PATH;
            source $HOME/.bash_profile
            source hack/git/.profile
            export GOPATH=$HOME/workspace/${jobName}

            make GO_VERSION=${env.BASE_GO_VERSION} \
                 MAVEN_DD_URL=${env.MAVEN_DD_URL}  \
                 SEMVER=${SEMVER} TAG=${env.BRANCH_SIMPLE_NAME} \
                 HUBS=${env.PLATFORM_DOCKER_REPO}/platform/kagent \
                 ART_HELM_URL=${env.ART_HELM_URL} \
                 ART_HELM_REPO=${env.ART_HELM_REPO} \
                 ART_MAVEN_DD_URL=${env.ART_MAVEN_DD_URL} \
                 ARTIF_CRED_USER=${ARTIF_CRED_USER} ARTIF_CRED_PASS=${ARTIF_CRED_PASS} \
                 NEXUS_CRED_USER=${NEXUS_CRED_USER} NEXUS_CRED_PASS=${NEXUS_CRED_PASS} \
                 release/charts
            """
            }
        }
    }
    stage('Run Email Script') {
        when {
            expression { env.SEMVER != SEMVER }
        }
        steps {
            script {
                sh '''
                cd $HOME_PATH;
                chmod +x ./email/auto-gen-email.sh
                ./email/auto-gen-email.sh
                '''
            }
        }
    }

    stage('Promote release bundle to Artifactory') {
//         when {
//             expression { env.SEMVER != SEMVER }
//         }
        steps {
            script {
                kubexArtifactorykagent(bundleVersion: "${SEMVER}", commitId: "${env.GIT_COMMIT_HASH}", componentName: "kagent", promotionTargetEnv: "DEV")
            }
        }
    }
 } //stages

  post {
    always {
        echo "Done"
    }
  }
}
