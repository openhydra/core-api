# 使用预先打包的 iso 一体机快速安装指南

本文档描述了如何使用预先打包好的镜像来安装操作已经 open-hydra 相关的组件

## 下载镜像

* 链接: https://pan.baidu.com/s/1qKXHVbcGrDScBNySPuY3uA?pwd=gvzr#list/path=%2Fopenhydra%2Frelease%2Frelease-2.0 提取码: gvzr 

## 最小硬件要求

* CPU core >= 8
* 250 GB 硬盘 >= 1
* memory >= 32 GB
* (可选)nvidia cuda gpu >= 1(driver: 535.183.01)

## usb 包含的组件介绍

* ubuntu-server-22.04-live-server-amd64.iso
  * k8s aio cluster deploy with [kubeclipper](https://github.com/kubeclipper/kubeclipper)
  * xinference -- 负责启动模型服务
    * 内置 -- Qwen1___5-4B-Chat-GPTQ-Int4，bge-large-zh-v1___5
  * aes-ai-tutor -- 负责只知识库和模型对话项目
  * ase-dashboard -- 负责提供前端页面
  * core-api -- 负责核心处理北向 api 和南向 api 通讯和权限授权等
  * open-hydra-server -- 负责启动 pod 服务
  * rancher.io/local-path -- 负责提供本地存储
  * keystone -- 负责提供用户认证和权限存储后端
  * prometheus -- 负责提供监控服务
  * mysql-operator -- 负责提供 mysql 数据库服务
  * sandbox 镜像 -- xedu 为内置镜像, 其他 paddle 、 keras 都需要等镜像下载
  * gpu-operator -- 管理 nvidia-gpu 的组件
    * nvidia-driver
    * nvidia-container-toolkit
    * nvidia-dcgm-exporter
    * nvidia-device-plugin
    * nvidia-cuda-validator

## 安装流程介绍

1. 插入 usb 到主机
2. 重启主机
3. 进入 boot 菜单(根据主板不同，进入方式不同，比如有些主板是按 F12，有些主板是按 F8 或者 F11)
4. 选择 usb 启动（这里可能会看到 usb 和 usb-partition 1 这是我们用 usb-partition 1）
5. 开始安装流程
6. 安装完成后，重启主机
7. 移除 usb
8. 进入系统
9. 等待完成

## 安装过程详细说明

### 步骤5--开始安装流程

* 进入 usb boot 界面后会有一端时间进行 iso 数据验证的过程大概需要持续3-5分钟不等，如下图

![installation](../images/installation-1.png)

* 语言选择--在弹出的语言学则界面我们这里选择 "English" 并按下回车键，如下图

![installation](../images/installation-2.png)

* 语言选择--在接下去弹出的对话框中直接按回车确认，如下图

![installation](../images/installation-3.png)

* 网络配置--在弹出的网络配置界，我们按 "键盘上" 键选择默认选中的第一张网卡，如果下图

![installation](../images/installation-4.png)

* 网络配置--按下回车键后，在弹出的网络配置界面，然后选择 "Edit IPv4" 后按下回车，如下图

![installation](../images/installation-6.png)

* 网络配置--在弹出的网络配置界面，我们选择 "Manual" 后按下回车，如下图

![installation](../images/installation-7.png)
![installation](../images/installation-8.png)

* 网络配置--在弹出的地址输入框中输入地址，我们这里为了后期网络配置方便我们设成 172.16.151.70，然后把光标指向 "Save" 按下回车，如下图

![installation](../images/installation-9.png)

* 网络配置--等待 10 秒后会出现 "Done" 选项在下方，我们按 "方向键下" 选中 "Done" 然后按回车，如下图

![installation](../images/installation-10.png)

* proxy 配置--跳过直接按回车，如下图

![installation](../images/installation-11.png)

* mirror 配置--跳过直接按回车，如下图

![installation](../images/installation-12.png)

* 升级页面--因为不通外部 intelnet 所有一段时间后会自动跳过
* 磁盘配置--我们把光标往上操作对准 "Set up this disk as an LVM group" 然后按下 "空格" 键，如下图

![installation](../images/installation-13.png)

* 磁盘配置--取消选择 "Set up this disk as an LVM group"，如下图

![installation](../images/installation-14.png)

* 磁盘配置--把光标对准下方的 "Done" 按下回车，如下图

![installation](../images/installation-15.png)

* 磁盘配置--在弹出的界面再次点击 "Done"，大家不用纠结涂上的磁盘大小，因为截图的机器不一定和你最终的磁盘大小一致，如下图

![installation](../images/installation-16.png)

* 磁盘配置--在弹出的界面，移动光标选择 "Continue" 然后按下回车确定，如下图

![installation](../images/installation-17.png)

* 用户配置--在弹出的界面，为了方便起见我们全部输入 "test" 然后操作光标对准 "Done" 然后按下回车，如下图

![installation](../images/installation-18.png)

* ssh配置--在弹出的是否安装 ssh 光标默认会对准 "Install OpenSSH server"，如下图

![installation](../images/installation-19.png)

* ssh配置--我们直接 "空格" 键，移动光标选择下方的 "Done" 然后按下回车确定，如下图

![installation](../images/installation-20.png)

* 等待安装完成，如下图

![installation](../images/installation-21.png)

* 等待出现 "Reboot Now" 选项时，移动光标选中 "Reboot now"

* 移除 usb 正常安装正常进入系统

![installation](../images/installation-22.jpeg)

## 查看安装输出日志

```bash
$ journalctl -u openhydra -f

# 输出
# 如果安装完成那么我们应该可以看到 maas.service: Succeeded 的输出
-- Logs begin at Sun 2023-12-10 10:58:29 UTC. --
...
Dec 10 11:29:14 test systemd[1]: openhydra.service: Succeeded.
Dec 10 11:29:14 test systemd[1]: Finished openhydra service.

# 检验 pod 的状态
$ kubectl get pod -A

# 输出，忽略 AGE 列，因为图是预先截取的
NAMESPACE                NAME                                                              READY   STATUS      RESTARTS        AGE
ai-education-studio      aes-ai-tutor-7fb494789-dv86c                                      1/1     Running     0               3h29m
ai-education-studio      aes-dashboard-deployment-89985c7c4-8z6dp                          1/1     Running     0               3h42m
ai-education-studio      core-api-server-6c8fbfb9f5-kkfc5                                  1/1     Running     0               3h42m
caas-monitoring-system   alertmanager-prometheus-kube-prometheus-alertmanager-0            2/2     Running     1 (4h5m ago)    4h5m
caas-monitoring-system   prometheus-caas-exporter-8688666679-c9kcr                         1/1     Running     0               4h6m
caas-monitoring-system   prometheus-kube-prometheus-blackbox-exporter-6cd7984856-b22qf     1/1     Running     0               4h6m
caas-monitoring-system   prometheus-kube-prometheus-operator-5c466bfcc8-5p7fq              1/1     Running     0               4h6m
caas-monitoring-system   prometheus-kube-state-metrics-597db47f5-tfrxx                     1/1     Running     0               4h6m
caas-monitoring-system   prometheus-node-exporter-cqndq                                    1/1     Running     0               4h6m
caas-monitoring-system   prometheus-prometheus-kube-prometheus-prometheus-0                3/3     Running     0               4h5m
caas-monitoring-system   strategy-controller-79676bd6dc-7h6xf                              1/1     Running     0               4h6m
caas-monitoring-system   thanos-bucketweb-7985489cb-4x2nw                                  1/1     Running     0               4h5m
caas-monitoring-system   thanos-compactor-74c4fcdff-85gd9                                  1/1     Running     0               4h5m
caas-monitoring-system   thanos-query-68bbcd6f6f-mrpzz                                     1/1     Running     0               4h5m
caas-monitoring-system   thanos-query-frontend-698cbdb6d7-2pq2v                            1/1     Running     0               4h5m
caas-monitoring-system   thanos-ruler-kube-prometheus-thanos-ruler-0                       2/2     Running     0               4h5m
caas-monitoring-system   thanos-storegateway-0                                             1/1     Running     0               4h5m
caas-system              caas-gateway-7c98694cfb-85d6s                                     1/1     Running     0               4h5m
caas-system              minio-78695dc855-6r6q7                                            1/1     Running     0               4h6m
caas                     caas-console-v1-d44984d67-wjpdd                                   1/1     Running     0               4h5m
caas                     caas-monitor-6fb677d5cf-gft9s                                     1/1     Running     4 (4h4m ago)    4h5m
caas                     caas-monitor-db-init-database-cz5x6                               0/1     Completed   4               4h5m
caas                     keystone-api-6844489c86-g4fgb                                     1/1     Running     0               4h5m
caas                     keystone-bootstrap-ft5gf                                          0/1     Completed   0               4h1m
caas                     keystone-credential-setup-xsds4                                   0/1     Completed   0               4h4m
caas                     keystone-db-init-wrgzq                                            0/1     Completed   0               4h3m
caas                     keystone-db-sync-kqr8h                                            0/1     Completed   0               4h3m
caas                     keystone-fernet-rotate-28788840-2kd89                             0/1     Completed   0               24m
caas                     keystone-fernet-setup-bjjld                                       0/1     Completed   0               4h3m
caas                     keystone-rabbit-init-6zw9s                                        0/1     Completed   0               4h3m
caas                     mariadb-galera-v1-0                                               2/2     Running     0               4h5m
caas                     memcached-0                                                       2/2     Running     0               4h5m
caas                     rabbitmq-0                                                        1/1     Running     0               4h5m
caas                     skyline-api-7bb74967d5-l2hmm                                      1/1     Running     0               4h5m
caas                     skyline-db-init-bftsh                                             0/1     Completed   0               4h5m
caas                     skyline-db-sync-c2gvv                                             0/1     Completed   0               4h5m
caas                     skyline-ks-endpoints-4bdg4                                        0/3     Completed   0               4h2m
caas                     skyline-ks-service-86hhw                                          0/1     Completed   0               4h3m
caas                     skyline-ks-user-kpvbq                                             0/1     Completed   0               4h2m
default                  init-pod                                                          1/1     Running     0               4h1m
gpu-operator             gpu-feature-discovery-xc7g8                                       2/2     Running     0               3h54m
gpu-operator             gpu-operator-1727317457-node-feature-discovery-gc-cb4745c866lw2   1/1     Running     0               3h59m
gpu-operator             gpu-operator-1727317457-node-feature-discovery-master-7556s7d22   0/1     Running     0               3h59m
gpu-operator             gpu-operator-1727317457-node-feature-discovery-worker-rm8fh       0/1     Running     0               3h59m
gpu-operator             gpu-operator-f965d7fc9-xpcvm                                      1/1     Running     0               3h59m
gpu-operator             nvidia-container-toolkit-daemonset-m5zl9                          1/1     Running     0               3h58m
gpu-operator             nvidia-cuda-validator-7sgbm                                       0/1     Completed   0               3h53m
gpu-operator             nvidia-dcgm-exporter-hzfx6                                        1/1     Running     0               3h58m
gpu-operator             nvidia-device-plugin-daemonset-t8j8s                              2/2     Running     0               3h54m
gpu-operator             nvidia-driver-daemonset-8qgdr                                     1/1     Running     0               3h59m
gpu-operator             nvidia-operator-validator-dzb94                                   1/1     Running     0               3h58m
ingress                  nginx-ingress-ingress-nginx-controller-t8xz8                      1/1     Running     6 (3h57m ago)   4h
kube-system              calico-kube-controllers-6f8667549b-xjcsf                          1/1     Running     0               4h7m
kube-system              calico-node-xbwkn                                                 1/1     Running     0               4h7m
kube-system              coredns-67f45c6654-6h9kb                                          1/1     Running     0               4h7m
kube-system              coredns-67f45c6654-986xd                                          1/1     Running     0               4h7m
kube-system              etcd-openhydra                                                    1/1     Running     0               4h8m
kube-system              kc-kubectl-98d9c6456-p9mmn                                        1/1     Running     0               4h7m
kube-system              kube-apiserver-openhydra                                          1/1     Running     0               4h8m
kube-system              kube-controller-manager-openhydra                                 1/1     Running     0               4h7m
kube-system              kube-proxy-rs9gn                                                  1/1     Running     0               4h7m
kube-system              kube-scheduler-openhydra                                          1/1     Running     0               4h8m
local-path-storage       local-path-provisioner-5bc7df5688-q6tvt                           1/1     Running     0               4h6m
mysql-operator           aes-ai-tutor-0                                                    2/2     Running     0               3h30m
mysql-operator           aes-ai-tutor-router-7c59b4c759-kmxp4                              1/1     Running     0               3h29m
mysql-operator           mycluster-0                                                       2/2     Running     0               3h43m
mysql-operator           mycluster-router-5d74f97d5b-4kqgj                                 1/1     Running     0               3h42m
mysql-operator           mysql-operator-66bfb7f6df-qtfq2                                   1/1     Running     0               3h43m
open-hydra               open-hydra-server-588db7b969-bg68r                                1/1     Running     0               3h42m
xinference               xinference-supervisor-86fd4fbd8-kqjhs                             1/1     Running     0               3h32m
xinference               xinference-worker-869898cc87-c55t5                                1/1     Running     0               3h32m
```

## 访问 dashboard

* 打开浏览器输入地址 `http://[ip]:30001`
  * 用户名/密码 -> aes-admin/YXNlLXBhc3N3b3Jk

## 屏幕唤醒黑屏的问题

* 测试有些带有 gpu 的服务器当你让服务器自动安装时，如果屏幕进入待机状态后唤醒后会出现花瓶现象，这是由于安装过程中同时安装了 gpu 驱动可能会导致显示有问题，不用担心可以重启一次即可，或者你可以直接 ssh 进行管理那么就不需要重启了

## 禁用自动升级

由于 gpu operator 是基于 linux 内核构建的，我们使用离线驱动程序镜像 `-generic` 来离线安装 gpu 驱动程序，这就是为什么每次 linux 内核升级时 gpu operator 将无法工作，因此我们需要禁用自动升级，除非您打算升级。

* 内核升级后带来的问题 -- gpu-operator 如果之前在离线环境中安装的，那么需要手动庚辛，反之 gpu-operator 理论上会自动更新并重新编译
* 关于 gpu-operator 无法识别中国特供卡的问题--请尝试更新 gpu-operator 到最新版本，如果还是无法识别请联系我们

```bash
# 禁止自动升级
$ apt-mark hold linux-headers-generic
$ apt-mark hold linux-image-generic
```
