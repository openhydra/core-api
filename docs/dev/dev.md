# core-api

It's project that act like a middle ware between the client and the server. It's main purpose is to provide a common interface for the client to interact with the server.

## Pull

Note this project container git submodules. To pull the project with all the submodules, use the following command:

```bash
# git pull code and submodules
$ git clone --recurse-submodules
```

## Before Commit

Before commit you code, make sure run unit test

```bash
# this will run unit test and go fmt and update api doc
$ make pre-commit
```

## Getting Started

### build the project

```bash
# build binary
$ make go-build
```

### run server

```bash
# create a config file
# note disableAuth should not be used in production environment
$ cat <<EOF > config.yaml
auth:
    keystone:
        endpoint: http://localhost:5000
        password: new-password
        domainId: default
        projectId: default
        tokenKeyInResponse: X-Subject-Token
        tokenKeyInRequest: X-Auth-Token
coreApi:
    port: "8080"
    disableAuth: true
EOF

# run it
$ cmd/core-api-server/
$ ./core-api-server -c config.yaml run
# output
./core-api-server --config /tmp/core-api-server-config.yaml run | lolcat                                                                                                                                                                                                                                                     3:33:18 PM
2024/07/16 15:33:24 INFO Starting server port=8080 disable_auth=true
      ___           ___           ___           ___           ___           ___                       ___           ___           ___           ___           ___           ___
     /\  \         /\  \         /\  \         /\  \         /\  \         /\  \          ___        /\  \         /\  \         /\  \         /\__\         /\  \         /\  \
    /::\  \       /::\  \       /::\  \       /::\  \       /::\  \       /::\  \        /\  \      /::\  \       /::\  \       /::\  \       /:/  /        /::\  \       /::\  \
   /:/\:\  \     /:/\:\  \     /:/\:\  \     /:/\:\  \     /:/\:\  \     /:/\:\  \       \:\  \    /:/\ \  \     /:/\:\  \     /:/\:\  \     /:/  /        /:/\:\  \     /:/\:\  \
  /:/  \:\  \   /:/  \:\  \   /::\~\:\  \   /::\~\:\  \   /::\~\:\  \   /::\~\:\  \      /::\__\  _\:\~\ \  \   /::\~\:\  \   /::\~\:\  \   /:/__/  ___   /::\~\:\  \   /::\~\:\  \
 /:/__/ \:\__\ /:/__/ \:\__\ /:/\:\ \:\__\ /:/\:\ \:\__\ /:/\:\ \:\__\ /:/\:\ \:\__\  __/:/\/__/ /\ \:\ \ \__\ /:/\:\ \:\__\ /:/\:\ \:\__\  |:|  | /\__\ /:/\:\ \:\__\ /:/\:\ \:\__\
 \:\  \  \/__/ \:\  \ /:/  / \/_|::\/:/  / \:\~\:\ \/__/ \/__\:\/:/  / \/__\:\/:/  / /\/:/  /    \:\ \:\ \/__/ \:\~\:\ \/__/ \/_|::\/:/  /  |:|  |/:/  / \:\~\:\ \/__/ \/_|::\/:/  /
  \:\  \        \:\  /:/  /     |:|::/  /   \:\ \:\__\        \::/  /       \::/  /  \::/__/      \:\ \:\__\    \:\ \:\__\      |:|::/  /   |:|__/:/  /   \:\ \:\__\      |:|::/  /
   \:\  \        \:\/:/  /      |:|\/__/     \:\ \/__/        /:/  /         \/__/    \:\__\       \:\/:/  /     \:\ \/__/      |:|\/__/     \::::/__/     \:\ \/__/      |:|\/__/
    \:\__\        \::/  /       |:|  |        \:\__\         /:/  /                    \/__/        \::/  /       \:\__\        |:|  |        ~~~~          \:\__\        |:|  |
     \/__/         \/__/         \|__|         \/__/         \/__/                                   \/__/         \/__/         \|__|                       \/__/         \|__|


# curl it
$ curl --location 'http://localhost:8080/apis/open-hydra-server.openhydra.io/v1/' -v 
```

### access api doc

```bash
# access api doc with curl or browser
$ curl --location 'http://localhost:8080/doc/routes.raml.html' -v
```

## development

### update api doc

```bash
# update api doc
# follow command will output both routes.raml and routes.raml.html in to asserts folder
# then these files will copy into image by future build
$ make update-api-doc
```

### build image

```bash
# add a comment for ci test
# build image
$ make image
```

### 基础用户名认证

* 在所有的请求中（除了 `/apis/core-api.openhydra.io/v1/users/login`） 都加上一个 header 他的 key 为 `Authorization`
* 这个 header 的值组成部分为 `Bearer base64([username]:[password])` 以下是举例说明

```bash
# 假设用户 user1 密码为 password 的在访问 api 时
# 那么以下是前端的处理逻辑
# 添加一个 header 在所有的请求中
const username = 'user1';
const password = 'password';

# 首先 base64 encode 用户名和密码
const base64Credentials = btoa(`${username}:${password}`);

# 组装成一个头
const authHeaderValue = `Bearer ${base64Credentials}`;

# 发送请求
fetch('http://localhost:8080/apis/your-api-endpoint', {
  method: 'GET', // or 'POST', 'PUT', 'DELETE', etc.
  headers: {
    'Authorization': authHeaderValue,
    // Add any other headers here
  },
  // Include other fetch options as needed
})
```

### 关于 group 设备标签

* 由于 openhydra 的能力有限，我们只利用 openhydra 标记 label 的能力，所以在前端请求是通过 `login` 会得到用户的 `group` 信息比如

```bash
{
    ... 其他信息
        "groups": [
            {
                "id": "dc447d6fa31e4b8caa95c5560af49208",
                "name": ""
            },
            {
              "id": "groupid2"
            }
        ],
}
```

* 在创建 devices 时，请加入 label 如下

```bash
{
    "kind": "Device",
    "apiVersion": "open-hydra-server.openhydra.io/v1",
    "metadata": {
        "name": "用户名",
        "labels": {
            "group": "dc447d6fa31e4b8caa95c5560af49208,groupid2"
        }
    },
    "spec": {
        "deviceType": "cpu",
        "openHydraUsername": "用户名",
        "sandboxName": "xedu"
    },
    "status": {}
}
```

* 获取 devices 时，可以通过 `group` 过滤

```bash
# 过滤班级
$ curl http://xxxxx/apis/open-hydra-server.openhydra.io/v1/devices?group=dc447d6fa31e4b8caa95c5560af49208&group=groupid2
```
