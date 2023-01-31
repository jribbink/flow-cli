/*
 * Flow CLI
 *
 * Copyright 2019 Dapper Labs, Inc.
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

package transactions

import (
	"strings"

	"github.com/onflow/flow-go-sdk"
	"github.com/spf13/cobra"

	"github.com/onflow/flow-cli/internal/command"
	"github.com/onflow/flow-cli/pkg/flowkit"
	"github.com/onflow/flow-cli/pkg/flowkit/services"
)

type flagsGet struct {
	Sealed  bool     `default:"true" flag:"sealed" info:"Wait for a sealed result"`
	Include []string `default:"" flag:"include" info:"Fields to include in the output. Valid values: signatures, code, payload."`
	Exclude []string `default:"" flag:"exclude" info:"Fields to exclude from the output. Valid values: events."`
}

var getFlags = flagsGet{}

var GetCommand = &command.Command{
	Cmd: &cobra.Command{
		Use:     "get <tx_id>",
		Aliases: []string{"status"},
		Short:   "Get the transaction by ID",
		Example: "flow transactions get 07a8...b433",
		Args:    cobra.ExactArgs(1),
	},
	Flags: &getFlags,
	Run:   get,
}

func get(
	args []string,
	_ flowkit.ReaderWriter,
	_ command.GlobalFlags,
	services *services.Services,
) (command.Result, error) {
	id := flow.HexToID(strings.TrimPrefix(args[0], "0x"))

	tx, result, err := services.Transactions.GetStatus(id, getFlags.Sealed)
	if err != nil {
		return nil, err
	}

	return &TransactionResult{
		result:  result,
		tx:      tx,
		include: getFlags.Include,
		exclude: getFlags.Exclude,
	}, nil
}
