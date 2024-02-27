// Copyright 2019 b<>com
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"github.com/sethvargo/go-envconfig"
	"github.com/tidwall/gjson"
	"gopkg.in/Graylog2/go-gelf.v2/gelf"
	"io"
	"log"
	"net/http"
	"time"
)

type envConfig struct {
	ListenOn    string `env:"LISTEN_ON,required"`
	GraylogAddr string `env:"GRAYLOG_ADDR,required"`
	HostId      string `env:"HOST_ID,required"`
}

func main() {
	var config envConfig
	var ctx context.Context

	if err := envconfig.Process(ctx, &config); err != nil {
		panic(err)
	}
	listenOnConfig := config.ListenOn
	graylogAddrConfig := config.GraylogAddr

	// Log service informations on startup
	log.Printf("Service is listening on: '%s'", listenOnConfig)
	log.Printf("Graylog server defined is: '%s'", graylogAddrConfig)

	gelfWriter, err := gelf.NewUDPWriter(graylogAddrConfig)
	if err != nil {
		log.Fatalf("gelf.NewWriter: %s", err)
	}
	defer gelfWriter.Close()

	log.Fatal(http.ListenAndServe(listenOnConfig, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// get payload
		promJSON, err := io.ReadAll(r.Body)
		// set payload as string
		spromJSON := string(promJSON)

		if err != nil {
			panic(err)
		}
		defer r.Body.Close()

		// Get alerts subset only
		result := gjson.Get(spromJSON, "alerts")
		// iterate over alerts
		result.ForEach(func(key, value gjson.Result) bool {
			log.Printf(value.String())
			msg := gelf.Message{
				Facility: "alertmanager2gelf",
				Version:  "1",
				Host:     config.HostId,
				Short:    gjson.Get(value.String(), "labels.alertname").String(),
				TimeUnix: float64(time.Now().Unix()),
				Extra: map[string]interface{}{
					"alertgroup": gjson.Get(value.String(), "labels.alertgroup").String(),
					"env":        gjson.Get(value.String(), "labels.env").String(),
					"host_name":  gjson.Get(value.String(), "labels.host_name").String(),
					"job":        gjson.Get(value.String(), "labels.job").String(),
					"severity":   gjson.Get(value.String(), "labels.severity").String(),
					"status":     gjson.Get(value.String(), "status").String(),
				},
			}

			err := gelfWriter.WriteMessage(&msg)
			if err != nil {
				log.Printf("gelf write: %s", err)
			}
			return true // keep iterating
		})

	})))
}
