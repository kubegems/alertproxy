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

type SingleAlertProxy interface {
	RenderRequest(oldReq *http.Request, alert Alert) (*http.Request, error)
	// if err not null, should retry
	CheckResponse(resp *http.Response) (bool, error)
}

type Alertproxy struct {
	*config.ProxyConfigs
	http.Client
}

func (p *Alertproxy) route() http.Handler {
	mux := mux.NewRouter()
	mux = mux.StrictSlash(true)
	mux.Methods(http.MethodPost).Path("/").HandlerFunc(p.HandelWebhook)
	return mux
}

func NewSingleAlertProxy(tpl *config.ProxyTemplate) SingleAlertProxy {
	tmpl := template.Must(template.New(tpl.Type).Parse(tpl.Template))
	switch tpl.Type {
	case alertProxyFeishu:
		return &FeishuRobot{
			Template: tmpl,
		}
	}
	return nil
}

func (p *Alertproxy) HandelWebhook(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	ptype := query.Get("type")
	var tpl *config.ProxyTemplate
	for _, v := range p.ProxyConfigs.Templates {
		if v.Type == ptype {
			tpl = v
		}
	}
	if tpl == nil {
		ResponseError(w, errors.Errorf("template: %s not found", ptype))
		return
	}

	alerts := WebhookAlert{}
	if err := json.NewDecoder(r.Body).Decode(&alerts); err != nil {
		ResponseError(w, errors.Wrap(err, "decode alerts"))
		return
	}
	sap := NewSingleAlertProxy(tpl)
	for _, alert := range alerts.Alerts {
		start := alert.StartsAt.In(time.Local)
		alert.StartsAt = &start
		req, err := sap.RenderRequest(r, alert)
		if err != nil {
			ResponseError(w, errors.Wrap(err, "render request"))
			return
		}

		if err := retry.Do(func() error {
			resp, err := p.Client.Do(req)
			if err != nil {
				log.Println(errors.Wrap(err, "do request"))
				return nil
			}

			shouldRetry, err := sap.CheckResponse(resp)
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
