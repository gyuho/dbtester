// Copyright 2017 CoreOS, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package dbtester

import (
	"bufio"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/coreos/etcd/clientv3"
	"golang.org/x/net/context"
)

func newPutEtcd3(conn clientv3.KV) ReqHandler {
	return func(ctx context.Context, req *request) error {
		_, err := conn.Do(ctx, req.etcdv3Op)
		return err
	}
}

// dialTotal counts the number of mustCreateConn calls so that endpoint
// connections can be handed out in round-robin order
var dialTotal int

func mustCreateConnEtcdv3(endpoints []string) *clientv3.Client {
	// endpoint := endpoints[dialTotal%len(endpoints)]
	// dialTotal++
	cfg := clientv3.Config{
		Endpoints: endpoints,
	}
	client, err := clientv3.New(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "dial error: %v\n", err)
		os.Exit(1)
	}
	return client
}

type etcdv3ClientCfg struct {
	totalConns   int64
	totalClients int64
}

func mustCreateClientsEtcdv3(endpoints []string, cfg etcdv3ClientCfg) []*clientv3.Client {
	conns := make([]*clientv3.Client, cfg.totalConns)
	for i := range conns {
		conns[i] = mustCreateConnEtcdv3(endpoints)
	}

	clients := make([]*clientv3.Client, cfg.totalClients)
	for i := range clients {
		clients[i] = conns[i%int(cfg.totalConns)]
	}
	return clients
}

func newGetEtcd3(conn clientv3.KV) ReqHandler {
	return func(ctx context.Context, req *request) error {
		_, err := conn.Do(ctx, req.etcdv3Op)
		return err
	}
}

func getTotalKeysEtcdv3(endpoints []string) map[string]int64 {
	rs := make(map[string]int64)
	for _, ep := range endpoints {
		if !strings.HasPrefix(ep, "http://") {
			ep = "http://" + ep
		}

		plog.Println("GET", ep+"/metrics")
		resp, err := http.Get(ep + "/metrics")
		if err != nil {
			plog.Println(err)
			rs[ep] = 0
		}
		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			txt := scanner.Text()
			if strings.HasPrefix(txt, "#") {
				continue
			}
			ts := strings.SplitN(txt, " ", 2)
			fv := 0.0
			if len(ts) == 2 {
				v, err := strconv.ParseFloat(ts[1], 64)
				if err == nil {
					fv = v
				}
			}
			if ts[0] == "etcd_debugging_mvcc_keys_total" {
				rs[ep] = int64(fv)
				break
			}
		}
		gracefulClose(resp)
	}

	plog.Println("getTotalKeysEtcdv3", rs)
	return rs
}
