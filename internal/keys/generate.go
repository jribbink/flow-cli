/*
 * Flow CLI
 *
 * Copyright 2019-2021 Dapper Labs, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package keys

import (
	"fmt"

	"github.com/onflow/flow-cli/internal/command"
	"github.com/onflow/flow-cli/pkg/flowcli/services"
	"github.com/spf13/cobra"
)

type flagsGenerate struct {
	Seed       string `flag:"seed" info:"Deterministic seed phrase"`
	KeySigAlgo string `default:"ECDSA_P256" flag:"sig-algo" info:"Signature algorithm"`
	Algo       string `default:"" flag:"algo" info:"⚠️ No longer supported: use sig-algo argument"`
}

var generateFlags = flagsGenerate{}

var GenerateCommand = &command.Command{
	Cmd: &cobra.Command{
		Use:     "generate",
		Short:   "Generate a new key-pair",
		Example: "flow keys generate",
	},
	Flags: &generateFlags,
	Run: func(
		cmd *cobra.Command,
		args []string,
		globalFlags command.GlobalFlags,
		services *services.Services,
	) (command.Result, error) {
		if generateFlags.Algo != "" {
			return nil, fmt.Errorf("⚠️ Algo flag no longer supported: use '--sig-algo' flag.")
		}

		privateKey, err := services.Keys.Generate(generateFlags.Seed, generateFlags.KeySigAlgo)
		if err != nil {
			return nil, err
		}

		pubKey := privateKey.PublicKey()
		return &KeyResult{privateKey: privateKey, publicKey: &pubKey}, nil
	},
}
