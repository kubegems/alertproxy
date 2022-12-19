# Alert Proxy

This project is an alert proxy for alertmanager's webhook alert.

We planed to support alert channels below:
- [x] Feishu Robot
- [x] Aliyun Msg
- [x] Aliyun Voice
- [x] Dingding Robot
- [ ] ...

## Getting Started

### Install Alert Proxy

1. Deploy
```bash
kubectl apply -f https://raw.githubusercontent.com/kubegems/alertproxy/main/bundle.yaml
```

2. Check
```bash
kubectl  get pod -n kubegems-monitoring
NAME                          READY   STATUS    RESTARTS   AGE
alertproxy-7d6cddbc96-brcr8   1/1     Running   0          1m
```

### Send alerts to alertproxy
Use feishu as example:
1. http api

`POST ${alertproxy_addr}?type=feishu&url=&{feishu_robot_addr}&at=${user_id}&signSecret=${sign_secret}`

body should be an alertmanager alert format, eg:
```json
{
    "receiver": "myreceiver",
    "status": "firing",
    "alerts": [
        {
            "status": "firing",
            "labels": {
                "cluster": "kubegems",
                "gems_alertname": "kubegems-test-alert",
                "gems_namespace": "kubegems-test-namespace",
                "severity": "error"
            },
            "annotations": {
                "message": "kubegems test alert message",
                "value": "0"
            },
            "startsAt": "2022-10-25T18:44:16.375635254+08:00",
            "endsAt": null,
            "generatorURL": "",
            "fingerprint": ""
        }
    ],
    "groupLabels": null,
    "commonLabels": null,
    "commonAnnotations": null,
    "externalURL": "",
    "version": "",
    "groupKey": "",
    "truncatedAlerts": 0
}
```

2. config in alertmanager

```yaml
  receivers:
  - name: feishu
    webhookConfigs:
    - sendResolved: false
      url: ${alertproxy_addr}?type=feishu&url=&{feishu_robot_addr}&at=${user_id}&signSecret=${sign_secret}
```

## Development

Refer to [DEVELOPMENT.md](DEVELOPMENT.md)
