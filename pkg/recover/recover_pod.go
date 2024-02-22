package recover

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/longxiucai/patrol-tools/pkg/common"
	"github.com/longxiucai/patrol-tools/pkg/ssh"

	"github.com/golang/glog"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
)

type ErrorPod struct {
	PodName      string
	PodNameSpace string
	Instance     string
}
type ErrorPodList []ErrorPod

func NewPodRecover(result interface{}) RecoverInterface {
	errListInterface, err := getErrorListFromResults(result, false)
	if err != nil {
		glog.Error(err)
		return nil
	}

	glog.Infof("Error pod list: %v", errListInterface)
	epl, ok := errListInterface.(ErrorPodList)
	if !ok {
		glog.Errorf("error converting to ErrorPodList slice")
		return nil
	}
	return epl
}

func shellPod(podName, namespace, cmd string, client kubernetes.Interface, sshconfig *common.SSHCONFIG) error {
	nodeName, nodeIP, err := getHostInfoByPodName(podName, namespace, client)
	if err != nil {
		return err
	}
	ip := net.ParseIP(nodeIP)
	sshClient, err := ssh.GetHostSSHClient(ip, sshconfig)
	if err != nil {
		return err
	}
	glog.Infof("Run command '%s' result: ", cmd)
	err = sshClient.CmdAsync(ip, cmd)
	if err != nil {
		return err
	}
	glog.Infof("[%s %s]Run command '%s' successfully", nodeName, nodeIP, cmd)
	return nil
}

func (epl ErrorPodList) Recover(client kubernetes.Interface, sshconfig *common.SSHCONFIG, action string) error {
	for _, errPod := range epl {
		switch action {
		case "delete":
			err := deletePod(errPod.PodName, errPod.PodNameSpace, client)
			if err != nil {
				return fmt.Errorf("delete pod error: %v", err)
			}
		case "restart":
			err := restartPod(errPod.PodName, errPod.PodNameSpace, client)
			if err != nil {
				return fmt.Errorf("restart pod error: %v", err)
			}
		default:
			err := shellPod(errPod.PodName, errPod.PodNameSpace, action, client, sshconfig)
			if err != nil {
				return fmt.Errorf("shell pod error: %v", err)
			}
		}
	}
	return nil
}

func watchPodDeletion(errPodName, errPodNamespace string, client kubernetes.Interface) error {
	watcher, err := client.CoreV1().Pods(errPodNamespace).Watch(context.Background(), metav1.ListOptions{
		FieldSelector: "metadata.name=" + errPodName,
	})
	if err != nil {
		glog.Error(err)
		return err
	}
	defer watcher.Stop()

	for {
		select {
		case event, ok := <-watcher.ResultChan():
			if !ok {
				return nil // Watch channel closed
			}
			if event.Type == watch.Deleted {
				// Pod deletion detected
				glog.Infof("Pod %s in namespace %s has been deleted\n", errPodName, errPodNamespace)
				return nil
			}
		case <-time.After(5 * time.Minute):
			// Timeout after 10 minutes if Pod deletion not detected
			glog.Warningf("Timeout[5 min] waiting for pod %s in namespace %s to be deleted\n", errPodName, errPodNamespace)
			return nil
		}
	}
}

func restartPod(podName, namespace string, client kubernetes.Interface) error {
	// 获取要重启的 Pod 对象
	pod, err := client.CoreV1().Pods(namespace).Get(context.TODO(), podName, metav1.GetOptions{})
	if err != nil {
		glog.Error(err)
		return err
	}

	// 更新 Pod 的注释来触发重启
	pod.Annotations["kubectl.kubernetes.io/restartedAt"] = time.Now().Format(time.RFC3339)
	pod.Annotations["kubectl.kubernetes.io/restartedBy"] = common.NAME

	// 更新 Pod 对象以执行重启
	_, err = client.CoreV1().Pods(namespace).Update(context.TODO(), pod, metav1.UpdateOptions{})
	if err != nil {
		glog.Error(err)
		return err
	}
	glog.Infof("Rebooted pod %s in %s", podName, namespace)
	return nil
}

func deletePod(podName, namespace string, client kubernetes.Interface) error {
	err := client.CoreV1().Pods(namespace).Delete(context.Background(), podName, metav1.DeleteOptions{})
	if err != nil {
		glog.Error(err)
		return err
	}
	glog.Infof("Deleting pod %s in %s", podName, namespace)
	return watchPodDeletion(podName, namespace, client)
}
