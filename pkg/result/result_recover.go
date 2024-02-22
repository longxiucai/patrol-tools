package result

import (
	"fmt"

	"github.com/longxiucai/patrol-tools/pkg/common"
	"github.com/longxiucai/patrol-tools/pkg/recover"

	"github.com/golang/glog"
	"k8s.io/client-go/kubernetes"
)

type ResultList []Result

type Result struct {
	common.Rule
	PromResult interface{}
}

func (rl ResultList) RunRecover(client kubernetes.Interface, sshconfig *common.SSHCONFIG) error {
	for _, result := range rl {
		if result.Recover.Enable {
			switch result.Recover.RecoveryType {
			case "service":
				esl := recover.NewServiceRecover(result.PromResult)
				if esl == nil {
					glog.Error("NewServiceRecover Error")
					return fmt.Errorf("NewServiceRecover Error")
				}
				err := esl.Recover(client, sshconfig, result.Recover.Action)
				if err != nil {
					return err
				}
			case "pod":
				epl := recover.NewPodRecover(result.PromResult)
				if epl == nil {
					glog.Error("NewPodRecover Error")
					return fmt.Errorf("NewPodRecover Error")
				}
				err := epl.Recover(client, sshconfig, result.Recover.Action)
				if err != nil {
					return err
				}
			default:
				return fmt.Errorf("unsupported type: %v", result.Recover.RecoveryType)
			}
		}
	}
	return nil
}
