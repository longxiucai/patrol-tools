# promql
```
      --auth-credentials string           optional auth credentials string for http requests to prometheus
      --auth-credentials-file string      optional path to an auth credentials file for http requests to prometheus
      --auth-type string                  optional auth scheme for http requests to prometheus e.g. "Basic" or "Bearer"
      --config string                     config file location (default promql.yaml)
      --end string                        query range end (either 'now', or an ISO 8601 formatted date string) (default "now")
      --host string                       prometheus server url (default "http://0.0.0.0:9090")
      --no-headers                        disable table headers for instant queries
      --output string                     override the default output format (graph for range queries, table for instant queries and metric names). Options: json,csv,excel (Cannot be used with --start)
      --output-path string                save to result path (default ".")
      --start string                      query range start duration (either as a lookback in h,m,s e.g. 1m, or as an ISO 8601 formatted date string). Required for range queries. Cannot be used with --output=excel
      --step string                       results step duration (h,m,s e.g. 1m) (default "1m")
      --time string                       time for instant queries (either 'now', or an ISO 8601 formatted date string) (default "now")
      --timeout int                       the timeout in seconds for all queries (default 10)
      --tls_config.ca_cert_file string    CA cert Path for TLS config
      --tls_config.cert_file string       client cert Path for TLS config
      --tls_config.insecure_skip_verify   disable the TLS verification of server certificates
      --tls_config.key_file string        client key for TLS config
      --tls_config.servername string      server name for TLS config
```
# 配置文件说明
```
host: "http://172.20.43.74:32090"
step: "1m"
start: ""
end: "now"
time: "now"
output: ""
output-path: "/home/lyx/desktop/"
no-headers: false
timeout: 10
kubeconfig: /home/lyx/.config/config
rules:
    - name: cpu
      expr: '100 - (avg by (instance) (rate(node_cpu_seconds_total{mode="idle"}[10m])) * 100)'
    - name: mem
      expr: '(1 - ((node_memory_MemFree_bytes + node_memory_Cached_bytes) / node_memory_MemTotal_bytes)) * 100 > 40'
      recover:
        type: pod
        action: "free -h"   # delete|restart|其他shell语句，脚本
        enable: true
    - name: disk
      expr: '(1 - (node_filesystem_avail_bytes{mountpoint="/"} / node_filesystem_size_bytes{mountpoint="/"})) * 100'
    - name: failed-pod
      expr: 'kube_pod_status_phase{phase!="Running",phase!="Succeeded"} == 1'
      recover:
        type: pod
        action: delete      # delete|restart|其他shell语句，脚本
        enable: false
    - name: pending-pod
      expr: 'kube_pod_status_phase{phase="Pending"} == 1'
      recover:
        type: pod
        action: delete      # delete|restart|其他shell语句，脚本
        enable: false
    - name : failed-service
        # expr: 'systemd_unit_state{type="service",state="failed"} == 1'
      expr: systemd_unit_state{type="service",state="failed",name="kylin-activation-check.service",pod="systemd-exporter-8vbj9"} == 1
      recover:
        type: service
        action: start       # start|restart|stop|disable|enable|其他shell语句，脚本
        enable: true
shell-rules: # 待开发
    - name: nginx-up
      command: docker ps -a |grep nginx-icbc|awk '{print $7}'
      operator: eq          # eq|gt|lt|ge|le| include（正则表达式 ？）
      value: Up             # 正则表达式 ？
      node-selector:
        - nginx
    - name: ks-up
      command: curl -XGET -uadmin:KylinSearchPassword "http://`kubectl get svc -n kcm kylinsearch-cluster-master|grep kylinsearch|awk '{print $3}'`:9200/_cluster/health"
      operator: include       # eq|gt|lt|ge|le| include（正则表达式 ？）
      value: "status":"green" # 正则表达式 ？
      node-selector:
        - master
ssh:
    user: root
    passwd: lyx@123444.    # 节点password全局配置
    port: 23
hosts:
  - ips:
      - 172.20.43.74              # 使用全局password配置
      - 172.20.43.73
      - 172.20.43.75
    role: master
    ssh:
      user: root
      passwd: lyx@123.     # 覆盖全局password配置
      port: 22
  - ips:   
      - 172.20.43.76
      - 172.20.43.77
      - 172.20.43.78
    role: worker
```
* 指定promethues的主机地址
```
host: "http://172.20.43.74:32090"
```
* 请求监控数据相关配置
```
step: "1m"
start: ""
end: "now"
time: "now"
timeout: 10
```
* 结果输出配置。为空的情况下，如果是range数据则输出为图形，instant数据则输出为表格打印到控制台。支持配置为json,csv,excel，其中excel不能为range数据，即start参数必须为空
```
output: ""
output-path: "/home/lyx/desktop/"
```
* k8s集群配置文件,即master节点中的~/.kube/config文件
```
kubeconfig: /home/lyx/.config/config
```
* 规则配置
```
rules:
    - name: cpu
      expr: '100 - (avg by (instance) (rate(node_cpu_seconds_total{mode="idle"}[10m])) * 100)'
```
* 治愈配置(pod资源)。示例为删除pending的pod,当填写其他shell语句时则会在对应异常节点执行对应的语句
```
rules:
    - name: pending-pod
      expr: 'kube_pod_status_phase{phase="Pending"} == 1'
      recover:
        type: pod
        action: delete      # delete|restart|其他shell语句（会在pod所在节点执行）
        enable: true
```
* 治愈配置(系统service资源)。示例为重启失败的服务，当填写其他shell语句时则会在对应异常节点执行对应的语句
```
rules:
    - name : failed-service
        expr: 'systemd_unit_state{type="service",state="failed"} == 1'
      recover:
        type: service
        action: restart       # start|restart|stop|disable|enable|其他shell语句（会在service所在节点执行）
        enable: true
```
* ssh配置
```
ssh:
    user: root
    passwd: lyx@123444.    # 节点password全局配置
    port: 23
hosts:
  - ips:
      - 172.20.43.74              # 使用全局password配置
      - 172.20.43.73
      - 172.20.43.75
    role: master
    ssh:
      user: root
      passwd: lyx@123.     # 覆盖全局password配置
      port: 22
  - ips:   
      - 172.20.43.76
      - 172.20.43.77
      - 172.20.43.78
    role: worker```