// Copyright 2021 Northern.tech AS
//
//    Licensed under the Apache License, Version 2.0 (the "License");
//    you may not use this file except in compliance with the License.
//    You may obtain a copy of the License at
//
//        http://www.apache.org/licenses/LICENSE-2.0
//
//    Unless required by applicable law or agreed to in writing, software
//    distributed under the License is distributed on an "AS IS" BASIS,
//    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//    See the License for the specific language governing permissions and
//    limitations under the License.

package http

import (
	"io/ioutil"

	"github.com/ant0ine/go-json-rest/rest"
	"github.com/mendersoftware/go-lib-micro/config"
	"github.com/mendersoftware/go-lib-micro/requestid"
	"github.com/mendersoftware/go-lib-micro/requestlog"
	"github.com/sirupsen/logrus"

	dconfig "github.com/mendersoftware/deployments/config"
)

func setUpRestTest(route string, routeType routerTypeHandler,
	handler func(w rest.ResponseWriter, r *rest.Request)) *rest.Api {

	router, _ := rest.MakeRouter(routeType(route, handler))
	api := rest.NewApi()
	api.Use(
		&requestlog.RequestLogMiddleware{
			BaseLogger: &logrus.Logger{Out: ioutil.Discard},
		},
		&requestid.RequestIdMiddleware{},
	)
	api.SetApp(router)

	return api
}

func init() {
	config.Config.SetDefault(dconfig.SettingAwsS3MaxImageSize, 10*1024*1024*1024)
}
