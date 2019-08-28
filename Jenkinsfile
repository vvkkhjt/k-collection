#!groovy
@Library('jk-pipeline-library') _

def imgHubQA = "hub.digi-sky.com"
def imgHubProd = "ccr.ccs.tencentyun.com"
def imgNamespace = 'yw'
def imgProdNamespace = 'digisky'
def artifactId = 'kapp-agent'

def getGitShortCommitId() {
    sh(returnStdout: true, script: 'git rev-parse --short HEAD').trim()
}

node("autonline"){
    stage("Checkout Code"){
        checkout scm
    }
    stage("Set PIPELINE VERSION"){
        env.PIPELINE_VERSION = "git${getGitShortCommitId()}.${env.BUILD_NUMBER}"
        echo "Set Pipeline Version: ${env.PIPELINE_VERSION}"
    }
    try{
        stage("Build Image"){
            container("dind"){
                imageName = "${imgNamespace}/${artifactId}:${env.PIPELINE_VERSION}"
                docker.withRegistry("https://${imgHubQA}", "imgbuild-harbor") {
                    harborImage = docker.build(imageName, ".")
                    harborImage.push()
                }
                docker.withRegistry("https://${imgHubProd}", "digisky-ops-ccr") {
                    prodImageName = "${imgHubProd}/${imgProdNamespace}/${artifactId}:${env.PIPELINE_VERSION}"
                    sh "docker tag ${imgHubQA}/${imageName} ${prodImageName}"
                    sh "docker push ${prodImageName}"
                }
            }
        }

        stage("Send Mail"){
            emailext body: "用户${env.BUILD_USER}刚刚完成了kapp-agent更新，请相关同事注意。更新信息地址: ${env.JOB_URL},版本号:${env.PIPELINE_VERSION}", recipientProviders: [[$class: 'DevelopersRecipientProvider']], subject: 'kapp-agent构建成功', to: 'wutao@digisky.com,wangzheying@digisky.com'
        }
    }catch(e){
        stage("Send Mail"){
            emailext body: "用户${env.BUILD_USER}刚刚完成了kapp-agent更新，请相关同事注意。更新信息地址: ${env.JOB_URL},版本号:${env.PIPELINE_VERSION},错误信息:${e}", recipientProviders: [[$class: 'DevelopersRecipientProvider']], subject: 'kapp-agent构建失败', to: 'wutao@digisky.com,wangzheying@digisky.com'
        }
    }
}
