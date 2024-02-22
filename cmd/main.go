package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/longxiucai/patrol-tools/pkg/clients"
	"github.com/longxiucai/patrol-tools/pkg/common"
	"github.com/longxiucai/patrol-tools/pkg/promql"
	"github.com/longxiucai/patrol-tools/pkg/shell"

	"github.com/golang/glog"
	"github.com/prometheus/common/config"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v2"
)

// cmd line args
var (
	pql promql.PromQL
	// query string
	// This is placeholder for the initial flag value. We ultimately parse it into the TimeoutDuration paramater of our config
	timeout int
	// timeStr is a placeholder for the inital "time" flag value. We parse it to a time.Time for use in our queries
	timeStr        string
	kubeconfigPath string
)
var stringFlags = []struct {
	// pflag.StringVar 更适合直接将标志的值与程序中的变量关联
	// pflag.String 更适合在后续使用 Viper 这样的配置管理工具时将标志值与配置绑定。
	name     string
	variable interface{} //暂时没作用，全部使用的pflag.String然后使用viper绑定，后面使用了viper.Unmarshal(&pql)载入配置文件  不是pflag.StringVar
	value    string
	desc     string
}{
	{"host", &pql.Host, "http://127.0.0.1:9090", "prometheus server url"},
	{"step", &pql.Step, "1m", "results step duration (h,m,s e.g. 1m)"},
	{"start", &pql.Start, "", "query range start duration (either as a lookback in h,m,s e.g. 1m, or as an ISO 8601 formatted date string). Required for range queries. Cannot be used with --output=excel"},
	{"end", &pql.End, "now", "query range end (either 'now', or an ISO 8601 formatted date string)"},
	{"time", &timeStr, "now", "time for instant queries (either 'now', or an ISO 8601 formatted date string)"},
	{"output", &pql.Output, "", "override the default output format (graph for range queries, table for instant queries and metric names). Options: json,csv,excel (Cannot be used with --start)"},
	{"auth-type", &pql.Auth.Type, "", "optional auth scheme for http requests to prometheus e.g. \"Basic\" or \"Bearer\""},
	{"auth-credentials", &pql.Auth.Credentials, "", "optional auth credentials string for http requests to prometheus"},
	{"auth-credentials-file", &pql.Auth.CredentialsFile, "", "optional path to an auth credentials file for http requests to prometheus"},
	{"tls_config.ca_cert_file", &pql.TLSConfig.CAFile, "", "CA cert Path for TLS config"},
	{"tls_config.cert_file", &pql.TLSConfig.CertFile, "", "client cert Path for TLS config"},
	{"tls_config.key_file", &pql.TLSConfig.KeyFile, "", "client key for TLS config"},
	{"tls_config.servername", &pql.TLSConfig.ServerName, "", "server name for TLS config"},
	{"output-path", &pql.OutputPath, ".", "save to result path"},
	{"kubeconfig", &pql.KubeConfigPath, "~/.kube/config", "kubeconfig for delete pod"},
}
var boolFlags = []struct {
	name     string
	variable *bool
	value    bool
	desc     string
}{
	{"no-headers", &pql.NoHeaders, false, "disable table headers for instant queries"},
	{"tls_config.insecure_skip_verify", &pql.TLSConfig.InsecureSkipVerify, false, "disable the TLS verification of server certificates"},
}

func main() {
	pqlConfigInit()
	// PrintStructAsKV(pql)
	resultList, warnings, err := pql.Run()
	if len(warnings) > 0 {
		glog.Warningf("Warnings: %v\n", warnings)
	}
	if err != nil {
		glog.Fatal(err)
	}

	// 读取配置文件
	yamlFile, err := ioutil.ReadFile(pql.CfgFile)
	if err != nil {
		log.Fatalf("Error reading YAML file: %v", err)
	}
	// ssh相关配置 处理异常资源
	var sshconfig common.SSHCONFIG
	err = yaml.Unmarshal(yamlFile, &sshconfig)
	if err != nil {
		glog.Fatalf("Error unmarshaling YAML: %v", err)
	}
	// shell巡检相关配置
	var shellconfig shell.SHELLCONFIG
	err = yaml.Unmarshal(yamlFile, &shellconfig)
	if err != nil {
		glog.Fatalf("Error unmarshaling YAML: %v", err)
	}

	shellconfig.Exec(&sshconfig)
	fmt.Println(shellconfig)
	// 输出检查结果，写入磁盘
	resultList.Write(pql.OutputPath, pql.Output, pql.NoHeaders)
	if err != nil {
		glog.Fatalln(err)
	}

	// kube client 处理异常资源
	cb, err := clients.NewBuilder(kubeconfigPath)
	if err != nil {
		glog.Errorf("creating clients error: %v", err)
	}
	client := cb.KubeClientOrDie("kcc-agent")

	// 处理异常资源
	err = resultList.RunRecover(client, &sshconfig)
	if err != nil {
		glog.Fatal(err)
	}

}

func init() {
	pflag.StringVar(&pql.CfgFile, "config", common.DEFAULTCONFIGFILE, "config file location")
	for _, flagConfig := range stringFlags {
		pflag.String(flagConfig.name, flagConfig.value, flagConfig.desc)
		if err := viper.BindPFlag(flagConfig.name, pflag.Lookup(flagConfig.name)); err != nil {
			glog.Fatalln(err)
		}
	}

	for _, flagConfig := range boolFlags {
		pflag.Bool(flagConfig.name, flagConfig.value, flagConfig.desc)
		if err := viper.BindPFlag(flagConfig.name, pflag.Lookup(flagConfig.name)); err != nil {
			glog.Fatalln(err)
		}
	}

	pflag.Int("timeout", 10, "the timeout in seconds for all queries")
	if err := viper.BindPFlag("timeout", pflag.Lookup("timeout")); err != nil {
		glog.Fatalln(err)
	}
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Parse()
	pflag.Lookup("logtostderr").Value.Set("true")
	viperInit()
}

func viperInit() {
	if pql.CfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(pql.CfgFile)
	} else {
		viper.SetConfigFile(common.DEFAULTCONFIGFILE)
	}
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.SetEnvPrefix(common.NAME)
	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			glog.Fatalf("Could not read config file: %v\n", err)
		}
	}
}

func PrintStructAsKV(s interface{}) {
	v := reflect.ValueOf(s)
	if v.Kind() == reflect.Struct {
		t := v.Type()
		for i := 0; i < v.NumField(); i++ {
			field := t.Field(i)
			value := v.Field(i)
			fmt.Printf("%s: %v\n", field.Name, value.Interface())
		}
	}
}

func pqlConfigInit() {
	// 载入配置文件
	viper.Unmarshal(&pql)
	if pql.Start != "" && pql.Output == "excel" {
		glog.Fatalln("Error args,--start Cannot be used with --output=excel,Or check config file")
	}

	// 赋值CA相关
	pql.Auth.Type = viper.GetString("auth-type")
	pql.Auth.Credentials = config.Secret(viper.GetString("auth-credentials"))
	pql.Auth.CredentialsFile = viper.GetString("auth-credentials-file")
	pql.TLSConfig = config.TLSConfig{
		CAFile:             viper.GetString("tls_config.ca_cert_file"),
		CertFile:           viper.GetString("tls_config.cert_file"),
		KeyFile:            viper.GetString("tls_config.key_file"),
		ServerName:         viper.GetString("tls_config.servername"),
		InsecureSkipVerify: viper.GetBool("tls_config.insecure_skip_verify"),
	}

	// Convert our timeout flag into a time.Duration
	timeout = viper.GetInt("timeout")
	pql.TimeoutDuration = time.Duration(int64(timeout)) * time.Second

	// Parse the timeStr from our --time flag if it was provided
	pql.Time = time.Now()
	now := pql.Time
	timeStr = viper.GetString("time")
	if timeStr != "now" {
		t, err := time.Parse(time.RFC3339, timeStr)
		if err != nil {
			glog.Fatalln(err)
		}
		pql.Time = t
	}

	// Create and set client interface
	cl, err := promql.CreateClientWithAuth(pql.Host, pql.Auth, pql.TLSConfig)
	if err != nil {
		glog.Fatalln(err)
	}
	pql.Client = cl

	// result
	dirTime := fmt.Sprintf("%4d%02d%02d-%02d%02d", now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute())
	pql.OutputPath = filepath.Join(viper.GetString("output-path"), "/result", dirTime)

	// mkdir result path
	if pql.Output == "json" || pql.Output == "csv" || pql.Output == "excel" {
		if err := os.MkdirAll(pql.OutputPath, 0755); err != nil {
			glog.Fatalf("mkdir path error: %s", err)
		}
	}

	//kube client
	kubeconfigPath = viper.GetString("kubeconfig")
}
