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
	"text/template"
	"time"

	"github.com/avast/retry-go"
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
			alertProxyMap[v.Type] = &FeishuRobot{Template: tmpl}
		default:
			log.Fatalf("unsupported alert proxy type: %s", v.Type)
		}
	}
}

type AlertProxy interface {
	// render a new http requets
	RenderRequest(oldReq *http.Request, alert Alert) (*http.Request, error)
	// return shouldRetry and error
	CheckResponse(resp *http.Response) (bool, error)
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
		req, err := p.RenderRequest(r, alert)
		if err != nil {
			ResponseError(w, errors.Wrap(err, "render request"))
			return
		}

		if err := retry.Do(func() error {
			resp, err := srv.Client.Do(req)
			if err != nil {
				log.Println(errors.Wrap(err, "do request"))
				return nil
			}

			shouldRetry, err := p.CheckResponse(resp)
			if shouldRetry {
				return errors.Wrap(err, "check response")
			}
			if err != nil {
				log.Println(err)
			} else {
				log.Printf("send alert to: %s, msg: %s", query.Get("url"), alert.Annotations["message"])
			}
			return nil
		}, retry.Attempts(5), retry.Delay(5*time.Second)); err != nil {
			log.Println(err)
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
