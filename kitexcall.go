/*
 * Copyright 2024 CloudWeGo Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
package main

import (
	"os"

	"github.com/kitex-contrib/kitexcall/pkg/argparse"
	"github.com/kitex-contrib/kitexcall/pkg/client"
	"github.com/kitex-contrib/kitexcall/pkg/config"
	"github.com/kitex-contrib/kitexcall/pkg/log"
)

var args = &argparse.Augment{}

func main() {

	if err := args.ParseArgs(); err != nil {
		log.Fail(err)
		os.Exit(1)
	}

	conf := args.BuildConfig()

	// After implementing server reflection,
	// options here can include list, describe, or invoke.

	var cli client.Client
	switch args.Type {
	case config.Thrift:
		cli = client.NewThriftGeneric()
	case config.Protobuf:
		// TODO
	}

	if err := client.InvokeRPC(cli, conf); err != nil {
		log.Fail(err)
		os.Exit(1)
	}

}
