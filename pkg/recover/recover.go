package recover

import (
	"fmt"

	"github.com/longxiucai/patrol-tools/pkg/common"

	"github.com/prometheus/common/model"
	"k8s.io/client-go/kubernetes"
)

type RecoverInterface interface {
	Recover(client kubernetes.Interface, sshconfig *common.SSHCONFIG, action string) error
}

func getErrorListFromResults(result interface{}, isErrorService bool) (interface{}, error) {
	var errpList ErrorPodList
	var errsList ErrorServiceList

	switch r := result.(type) {
	case model.Vector:
		for _, v := range r {
			if isErrorService {
				err := ErrorService{
					ServiceName: string(v.Metric["name"]),
					Instance:    string(v.Metric["instance"]),
					PodName:     string(v.Metric["pod"]),
				}
				errsList = append(errsList, err)
			} else {
				err := ErrorPod{
					PodName:      string(v.Metric["pod"]),
					PodNameSpace: string(v.Metric["namespace"]),
					Instance:     string(v.Metric["instance"]),
				}
				errpList = append(errpList, err)
			}
		}
	case model.Matrix:
		for _, m := range r {
			if isErrorService {
				err := ErrorService{
					ServiceName: string(m.Metric["name"]),
					Instance:    string(m.Metric["instance"]),
					PodName:     string(m.Metric["pod"]),
				}
				errsList = append(errsList, err)
			} else {
				err := ErrorPod{
					PodName:      string(m.Metric["pod"]),
					PodNameSpace: string(m.Metric["namespace"]),
					Instance:     string(m.Metric["instance"]),
				}
				errpList = append(errpList, err)
			}
		}
	default:
		return nil, fmt.Errorf("unsupported result type: %v", r)
	}
	if isErrorService {
		return errsList, nil
	} else {
		return errpList, nil
	}
}
