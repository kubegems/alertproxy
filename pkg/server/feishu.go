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
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"text/template"
	"time"

	"github.com/pkg/errors"
)

type FeishuRobot struct {
	*template.Template
	URL        string
	At         []string
	SignSecret string
}

type FeishuResp struct {
	// success
	StatusCode    int    `json:"StatusCode"`
	StatusMessage string `json:"StatusMessage"`

	// failed
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

func (resp FeishuResp) String() string {
	bts, _ := json.Marshal(resp)
	return string(bts)
}

// copy from feishu
func GenSign(secret string, timestamp int64) (string, error) {
	//timestamp + key 做sha256, 再进行base64 encode
	stringToSign := fmt.Sprintf("%v", timestamp) + "\n" + secret
	var data []byte
	h := hmac.New(sha256.New, []byte(stringToSign))
	_, err := h.Write(data)
	if err != nil {
		return "", err
	}
	signature := base64.StdEncoding.EncodeToString(h.Sum(nil))
	return signature, nil
}

func (f FeishuRobot) RenderRequest(oldReq *http.Request, alert Alert) (*http.Request, error) {
	query := oldReq.URL.Query()
	f.URL = query.Get("url")
	if query.Get("at") != "" {
		f.At = strings.Split(query.Get("at"), ",")
	}

	obj := struct {
		Alert
		// for template
		At        []string
		Timestamp int64
		Sign      string
	}{
		Alert: alert,
		At:    f.At,
	}

	f.SignSecret = query.Get("signSecret")
	if f.SignSecret != "" {
		obj.Timestamp = time.Now().Unix()
		sign, err := GenSign(f.SignSecret, obj.Timestamp)
		if err != nil {
			return nil, errors.Wrap(err, "gen sign")
		}
		obj.Sign = sign
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

// {"StatusCode":0,"StatusMessage":"","code":9499,"msg":"too many request"}
// {"StatusCode":0,"StatusMessage":"","code":19007,"msg":"Bot Not Enabled"}
// {"StatusCode":0,"StatusMessage":"","code":19021,"msg":"sign match fail or timestamp is not within one hour from current time"}
func (f FeishuRobot) CheckResponse(resp *http.Response) (shouldRetry bool, err error) {
	obj := FeishuResp{}
	bts, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, err
	}
	if err := json.Unmarshal(bts, &obj); err != nil {
		return false, err
	}
	if obj.StatusCode != 0 || obj.Code != 0 {
		if obj.Code == 9499 { // too many request
			return true, errors.New(obj.String())
		} else {
			return false, errors.New(obj.String())
		}
	}
	return false, nil
}
