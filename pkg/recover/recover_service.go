package recover

import (
	"context"
	"fmt"
	"net"

	"github.com/longxiucai/patrol-tools/pkg/common"
	"github.com/longxiucai/patrol-tools/pkg/ssh"

	"github.com/golang/glog"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type ErrorService struct {
	ServiceName string
	PodName     string
	Instance    string
}

type ErrorServiceList []ErrorService

func NewServiceRecover(result interface{}) RecoverInterface {
	errListInterface, err := getErrorListFromResults(result, true)
	if err != nil {
		glog.Error(err)
		return nil

	}
	glog.Infof("Error service list: %v", errListInterface)
	esl, ok := errListInterface.(ErrorServiceList)
	if !ok {
		glog.Errorf("error converting to ErrorServiceList slice")
		return nil
	}
	return esl
}

func (esl ErrorServiceList) Recover(client kubernetes.Interface, sshconfig *common.SSHCONFIG, action string) error {
	for _, errService := range esl {
		err := doServiceAction(errService, action, sshconfig, client)
		if err != nil {
			glog.Error(err)
			return fmt.Errorf("restart service error: %v", err)
		}
	}
	return nil
}
func getHostInfoByPodName(podname, namespace string, client kubernetes.Interface) (string, string, error) {
	PodList, err := client.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return "", "", err
	}
	var nodeName, nodeIP string
	// 遍历所有 Pod 查找匹配的 Pod 获取所在节点以及节点ip
	for _, pod := range PodList.Items {
		if pod.Name == podname {
			nodeName = pod.Spec.NodeName
			nodeIP = pod.Status.HostIP
			glog.Infof("Pod %s running on Node %s %s", podname, nodeName, nodeIP)
		}
	}
	return nodeName, nodeIP, nil
}
func doServiceAction(serviceResult ErrorService, action string, sshconfig *common.SSHCONFIG, client kubernetes.Interface) error {
	nodeName, nodeIP, err := getHostInfoByPodName(serviceResult.PodName, "", client)
	if err != nil {
		return err
	}
	ip := net.ParseIP(nodeIP)
	sshClient, err := ssh.GetHostSSHClient(ip, sshconfig)
	if err != nil {
		return err
	}

	var cmd string
	if action == "start" || action == "stop" || action == "restart" || action == "disable" || action == "enable" {
		cmd = fmt.Sprintf("systemctl %s %s", action, serviceResult.ServiceName)
	} else {
		cmd = action
	}
	glog.Infof("run command '%s' result: ", cmd)
	err = sshClient.CmdAsync(ip, cmd)
	if err != nil {
		return err
	}
	glog.Infof("[%s %s]Run command '%s' successfully", nodeName, nodeIP, cmd)
	return nil
}
