package common

import (
	"fmt"
	"net"
	"os"
	"time"

	"github.com/xuri/excelize/v2"
)

type Rule struct {
	Name    string  `mapstructure:"name" yaml:"name"`
	Expr    string  `mapstructure:"expr" yaml:"expr"`
	Recover Recover `mapstructure:"recover" yaml:"recover"`
}
type Recover struct {
	RecoveryType string `mapstructure:"type" yaml:"type"`
	Action       string `mapstructure:"action" yaml:"action"`
	Enable       bool   `mapstructure:"enable" yaml:"enable"`
}

const (
	NAME              = "patrol"
	OUTPUTFILEPREFIX  = NAME + "_prometheus_data"
	DEFAULTCONFIGFILE = NAME + ".yaml"
)

const (
	ROOT = "root"
)

type SSH struct {
	Encrypted bool   `mapstructure:"encrypted" yaml:"encrypted,omitempty"`
	User      string `mapstructure:"user" yaml:"user,omitempty"`
	Passwd    string `mapstructure:"passwd" yaml:"passwd,omitempty"`
	Pk        string `mapstructure:"pk" yaml:"pk,omitempty"`
	PkPasswd  string `mapstructure:"pkPasswd" yaml:"pkPasswd,omitempty"`
	Port      string `mapstructure:"port" yaml:"port,omitempty"`
}

type Host struct {
	IPS   []net.IP `mapstructure:"ips" yaml:"ips,omitempty"`
	Roles []string `mapstructure:"roles" yaml:"roles,omitempty"`
	//overwrite SSH config
	SSH `mapstructure:"ssh" yaml:"ssh,omitempty"`
	//overwrite env
	Env []string `mapstructure:"env" yaml:"env,omitempty"`
}

type SSHCONFIG struct {
	Hosts []Host `mapstructure:"hosts" yaml:"hosts"`
	SSH   `mapstructure:"ssh" yaml:"ssh,omitempty"`
}

type ExcelFile struct {
	FullPath  string
	SheetName string
	ExcelLize *excelize.File
}

func NewExcelFile(fullPath string, outputTime time.Time) (ExcelFile, error) {
	var excel ExcelFile
	var excelizer *excelize.File
	var err error
	if _, openErr := os.Stat(fullPath); os.IsNotExist(openErr) { // 如果文件不存在，创建一个新文件
		excelizer = excelize.NewFile()
	} else { // 否则，打开现有文件
		excelizer, err = excelize.OpenFile(fullPath)
		if err != nil {
			return excel, err
		}
	}
	excel = ExcelFile{
		FullPath:  fullPath,
		SheetName: fmt.Sprintf("%02d时%02d分%02d秒", outputTime.Hour(), outputTime.Minute(), outputTime.Second()),
		ExcelLize: excelizer,
	}
	return excel, err
}
