/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package server

import (
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"strings"
	"text/template"
	"time"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"kubegems.io/alertproxy/config"
)

var alertProxyMap = map[config.ProxyType]AlertProxy{}

func Init(cfgs *config.ProxyConfigs) {
	for _, v := range cfgs.Templates {
		if _, ok := alertProxyMap[v.Type]; ok {
			log.Fatalf("duplicated alert proxy type: %s", v.Type)
		}
		tmpl := template.Must(template.New(string(v.Type)).Parse(v.Template))
		switch v.Type {
		case config.Feishu:
			alertProxyMap[v.Type] = NewFeishuRobot(tmpl)
		case config.AliyunMsg:
			alertProxyMap[v.Type] = NewAliyunMsg(tmpl)
		case config.AliyunVoice:
			alertProxyMap[v.Type] = NewAliyunVoice(tmpl)
		default:
			log.Fatalf("unsupported alert proxy type: %s", v.Type)
		}
	}
}

type AlertProxy interface {
	// Do a new alert requet
	DoRequest(params url.Values, alert Alert) error
}

type AlertproxyServer struct {
	http.Client
}

func (p *AlertproxyServer) route() http.Handler {
	mux := mux.NewRouter()
	mux = mux.StrictSlash(true)
	mux.Methods(http.MethodPost).Path("/").HandlerFunc(p.HandelWebhook)
	return mux
}

func (srv *AlertproxyServer) HandelWebhook(w http.ResponseWriter, r *http.Request) {
	alerts := WebhookAlert{}
	if err := json.NewDecoder(r.Body).Decode(&alerts); err != nil {
		ResponseError(w, errors.Wrap(err, "decode alerts"))
		return
	}

	query := r.URL.Query()
	ptype := query.Get("type")
	p, ok := alertProxyMap[config.ProxyType(ptype)]
	if !ok {
		ResponseError(w, errors.Errorf("proxy type: %s not found", ptype))
		return
	}

	for _, alert := range alerts.Alerts {
		start := alert.StartsAt.In(time.Local)
		alert.StartsAt = &start
		if msg, ok := alert.Annotations["message"]; ok {
			// 为 " 转义
			alert.Annotations["message"] = strings.ReplaceAll(msg, `"`, `\"`)
		}
		if err := p.DoRequest(r.URL.Query(), alert); err != nil {
			ResponseError(w, errors.Wrapf(err, "do request by %s", ptype))
			return
		}
	}
	ResponseOK(w, "ok")
}

type WebhookAlert struct {
	Receiver          string            `json:"receiver"`
	Status            string            `json:"status"`
	Alerts            []Alert           `json:"alerts"`
	GroupLabels       map[string]string `json:"groupLabels"`
	CommonLabels      map[string]string `json:"commonLabels"`
	CommonAnnotations map[string]string `json:"commonAnnotations"`
	ExternalURL       string            `json:"externalURL"`
	Version           string            `json:"version"`
	GroupKey          string            `json:"groupKey"`
	TruncatedAlerts   int64             `json:"truncatedAlerts"`
}

type Alert struct {
	Status       string            `json:"status"`
	Labels       map[string]string `json:"labels"`
	Annotations  map[string]string `json:"annotations"`
	StartsAt     *time.Time        `json:"startsAt"`
	EndsAt       *time.Time        `json:"endsAt"`
	GeneratorURL string            `json:"generatorURL"`
	Fingerprint  string            `json:"fingerprint"`
}
