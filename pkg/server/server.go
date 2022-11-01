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
	"context"
	"log"
	"net"
	"net/http"
	"time"

	"kubegems.io/alertproxy/config"
)

func Run(ctx context.Context, opts *config.ProxyConfigs) error {
	Init(opts)
	proxy := AlertproxyServer{
		Client: http.Client{Timeout: 10 * time.Second},
	}
	server := http.Server{
		Addr: opts.Listen,
		BaseContext: func(l net.Listener) context.Context {
			return ctx
		},
		Handler: proxy.route(),
	}
	go func() {
		<-ctx.Done()
		server.Shutdown(ctx)
	}()
	log.Printf("alertproxy serve on: %s", opts.Listen)
	return server.ListenAndServe()
}
