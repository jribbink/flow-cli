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

package flowkit

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/onflow/cadence"
	jsoncdc "github.com/onflow/cadence/encoding/json"
	"github.com/onflow/cadence/runtime"
	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/cmd"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
)

type CadenceArgument struct {
	Value cadence.Value
}

func (v CadenceArgument) MarshalJSON() ([]byte, error) {
	return jsoncdc.Encode(v.Value)
}

func (v *CadenceArgument) UnmarshalJSON(b []byte) (err error) {
	v.Value, err = jsoncdc.Decode(nil, b)
	if err != nil {
		return err
	}
	return nil
}

func ParseArgumentsJSON(input string) ([]cadence.Value, error) {
	var args []CadenceArgument
	b := []byte(input)
	err := json.Unmarshal(b, &args)

	if err != nil {
		return nil, err
	}

	cadenceArgs := make([]cadence.Value, len(args))
	for i, arg := range args {
		cadenceArgs[i] = arg.Value
	}
	return cadenceArgs, nil
}

// sanitizeAddressArg sanitize address and make sure it has 0x prefix
func processValue(argType string, argValue string) interface{} {
	if argType == "Address" && !strings.Contains(argValue, "0x") {
		return fmt.Sprintf("0x%s", argValue)
	} else if argType == "Bool" {
		converted, _ := strconv.ParseBool(argValue)
		return converted
	}

	return argValue
}

func GetAuthorizerCount(fileName string, code []byte) int {

	codes := make(map[common.Location][]byte)
	location := common.StringLocation(fileName)
	program, _ := cmd.PrepareProgram(code, location, codes)

	transactionDeclaration := program.TransactionDeclarations()
	if len(transactionDeclaration) == 1 {
		if transactionDeclaration[0].Prepare != nil {
			parameters := transactionDeclaration[0].Prepare.FunctionDeclaration.ParameterList
			if parameters != nil {
				return len(parameters.Parameters)
			}
		}
	}
	return 0
}

func ParseArgumentsWithoutType(fileName string, code []byte, args []string) (scriptArgs []cadence.Value, err error) {

	resultArgs := make([]cadence.Value, 0, len(args))

	codes := map[common.Location][]byte{}
	location := common.StringLocation(fileName)
	program, must := cmd.PrepareProgram(code, location, codes)
	checker, _ := cmd.PrepareChecker(program, location, codes, nil, must)

	var parameterList []*ast.Parameter

	functionDeclaration := sema.FunctionEntryPointDeclaration(program)
	if functionDeclaration != nil {
		if functionDeclaration.ParameterList != nil {
			parameterList = functionDeclaration.ParameterList.Parameters
		}
	}

	transactionDeclaration := program.TransactionDeclarations()
	if len(transactionDeclaration) == 1 {
		if transactionDeclaration[0].ParameterList != nil {
			parameterList = transactionDeclaration[0].ParameterList.Parameters
		}
	}

	contractDeclaration := program.SoleContractDeclaration()
	if contractDeclaration != nil {
		contractInitializer := contractDeclaration.Members.Initializers()
		if len(contractInitializer) == 1 {
			if contractInitializer[0].FunctionDeclaration.ParameterList != nil {
				parameterList = contractInitializer[0].FunctionDeclaration.ParameterList.Parameters
			}
		}
	}

	if parameterList == nil {
		return resultArgs, nil
	}

	if len(parameterList) != len(args) {
		return nil, fmt.Errorf("argument count is %d, expected %d", len(args), len(parameterList))
	}

	inter, err := interpreter.NewInterpreter(nil, nil, &interpreter.Config{})
	if err != nil {
		return nil, err
	}

	for index, argumentString := range args {
		astType := parameterList[index].TypeAnnotation.Type
		semaType := checker.ConvertType(astType)

		for {
			switch v := semaType.(type) {
			case *sema.OptionalType:
				semaType = v.Type
				continue

			case *sema.SimpleType:
				if v == sema.StringType {
					if len(argumentString) > 0 && !strings.HasPrefix(argumentString, "\"") {
						argumentString = "\"" + argumentString + "\""
					}
				}

			case *sema.AddressType:
				if !strings.Contains(argumentString, "0x") {
					argumentString = fmt.Sprintf("0x%s", argumentString)
				}
			}
			break
		}

		var value, err = runtime.ParseLiteral(argumentString, semaType, inter)
		if err != nil {
			return nil, fmt.Errorf(
				"argument `%s` is not expected type `%s`",
				parameterList[index].Identifier,
				semaType.QualifiedString(),
			)
		}
		resultArgs = append(resultArgs, value)
	}
	return resultArgs, nil
}
