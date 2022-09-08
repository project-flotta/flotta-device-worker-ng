# Flotta device worker


This is an _unofficial_ implementation of the device-worker for [Project Flotta](https://project-flotta.io/).

## Motivation

The current implementation of device-worker does not take into consideration the resources available on the device _while_ running workloads.
This implementation aims to have a high resilience by ensuring that _workloads_ do not deplete the device of resources.
Another motivation is to make the device-worker agnostic about the work to do. The device-worker should only manage and run workloads without any other additional work like monitoring, logging, or data collection. 
All these additional tasks could be run like any other workloads.

Last but not least, this implementation *does not* use _yggdrasil_ as broker. It has a simple implementation of _yggdrasil_ API but it is a standalone in this regard.

## Differences between the official agent and device-worker-ng

#### 1. Workload Kind
Currently, the official `device-worker` supports only two types of workloads: pods and ansible. 

The `device-worker-ng` supports `pod` and, in addition to `device-worker`, supports the `k8s` workloads.


#### 2. Run workloads as `rootfull` or `rootless`
Workload CR has a new field `Rootless` which set the type of executor which will be used to run the workload. In case of **podman**, the pods will be either `rootless` or `rootless` based on this field. To be able to run `rootfull` pods, the agent must be run as *root* and to run `rootless` pods the `XDG_RUNTIME_DIR` must be provided in the `config.yaml` file.


#### 3. OS monitoring and logging
`device-worker-ng` does not intend to have any build-in OS monitoring and logging.

#### 4. Cron
You can define a workload to run as `cron` by adding a field in the CR of the workload:
```yaml
apiVersion: management.project-flotta.io/v1alpha1
kind: EdgeWorkload
metadata:
  name: workload2
spec:
  deviceSelector:
    matchLabels:
      device: toto
  type: pod
  cron: *\1 * * * *
  pod:
    spec:
      containers:
        - name: work
          image: quay.io/ctupangiu/cron-workload:latest
```
> For k8s workloads, the cron will be ignored.

This manifest adds a `cron` definition to the workload making the workload to be executed every minute.

#### 5. Retry
Because `device-worker-ng` does not rely on `systemd` to restart a failling job, it has its own retry mechanism. It supports `constant` or `exponential` retry intervals. 
Currently, there is no way to configure the interval though the CR of the workload. The retry is set to `20s` constant interval.


#### 6. EdgeDevice's profiles
Profiles are a new feature of `device-worker-ns`. The idea behind it is to allow workloads to run only when certains profiles are active.
Each profile has a set of conditions which, basically, are boolean expression like: `cpu > 23%` or `cpu < 5% || cpu > 40%`. 
The value for variables can be sent to the device though a POST request like:
```
curl -X -H "Content-Type: application/json" -v -d '{"value":20,"name":"cpu"}' http://localhost:8080/metrics
```
The endpoint is hard coded to `http://localhost:8080/metrics`.
After each `POST`, the scheduler will evaluate each condition for each profile and determinate which if the job will run or not.
An `EdgeDevice` CR with profiles can look like this:
```yaml
apiVersion: management.project-flotta.io/v1alpha1
kind: EdgeDevice
metadata:
  labels:
    app: camera
    name: toto
    namespace: default
spec:
  heartbeat:
    periodSeconds: 2
    profiles:
      - name: test
        conditions:
          - name: "off"
            expression: "x<3% || x == nil"
          - name: "another_one"
            expression: "y>3"
```
This CR defines one profile named `test` whith two conditions `off` and `another_one`. 
To enable profiles for a workload, we need to set this profile in the CR of the workload:
```yaml
apiVersion: management.project-flotta.io/v1alpha1
kind: EdgeWorkload
metadata:
  name: nginx-rootfull
spec:
  deviceSelector:
    matchLabels:
      app: camera
    profiles:
    	- name: "test"
        conditions:
          - "off"
  type: k8s
  pod:
    spec:
      containers:
        - name: nginx1
          image: quay.io/project-flotta/nginx:1.21.6
```
For this workload, we add only one condition `off` so, if that condition is `true`, the executor will run the workload.

## Prerequisites

You must have flotta operator and API server running and client certificates already generated.
For more information, please read the [documentation](https://project-flotta.io/documentation/latest/intro/overview.html).

**Remark:**

To have a support for `profiles`, `retry` and `cron` feature, you may need to use this repo [github.com/tupyy/k4e-operator](github.com/tupyy/k4e-operator) of `flotta-edge-api` and `flotta-controller` because, curently, the official repo does not support those feature:


## Build
> You need go1.18 installed to be able to build the project.

Build:
```shell
make vendor
make build
```

## Usage

Create a configuration yaml like:
```
LOG_LEVEL: debug
CA_ROOT: /home/cosmin/projects/device-worker-ng/resources/certificates/ca.pem
CERT: /home/cosmin/projects/device-worker-ng/resources/certificates/cert.pem
KEY: /home/cosmin/projects/device-worker-ng/resources/certificates/key.pem
SERVER: https://127.0.0.1:8043
XDG_RUNTIME_DIR: /run/user/1000
KUBECONFIG: /path/to/kubeconfig/if/any
DEVICE_ID: toto
```

Run the device-worker:

```
device-worker-ng --config config.yaml 
```
> If you want to execute workloads on `rootfull podman`, you will need to run the `device-worker-ng` as root.

If you prefer, you can use environment variable with the prefix `EDGE_DEVICE`:
```
EDGE_DEVICE_CA_ROOT=path_to_ca EDGE_DEVICE_CERT=path_to_cert EDGE_DEVICE_KEY=path_to_key EDGE_DEVICE_SERVER=https://127.0.0.1:8043 device-worker-ng
```


