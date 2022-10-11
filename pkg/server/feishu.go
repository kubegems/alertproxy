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
	"bytes"
	"net/http"
	"strings"
	"text/template"
)

type FeishuRobot struct {
	*template.Template
	URL string
	At  []string
}

const alertProxyFeishu = "feishu"

func (f *FeishuRobot) RenderRequest(oldReq *http.Request, alert Alert) (*http.Request, error) {
	query := oldReq.URL.Query()
	f.URL = query.Get("url")
	f.At = strings.Split(query.Get("at"), ",")

	obj := struct {
		Alert
		At []string
	}{
		Alert: alert,
		At:    f.At,
	}

	buf := bytes.NewBuffer([]byte{})
	if err := f.Template.Execute(buf, obj); err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPost, f.URL, buf)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	return req, nil
}
