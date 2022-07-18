/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2022 Dapper Labs, Inc.
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

package runtime

// TODO:
//func TestRuntimeCoverage(t *testing.T) {
//
//	t.Parallel()
//
//	runtime := newTestInterpreterRuntime()
//
//	importedScript := []byte(`
//      pub fun answer(): Int {
//        var i = 0
//        while i < 42 {
//          i = i + 1
//        }
//        return i
//      }
//    `)
//
//	script := []byte(`
//      import "imported"
//
//      pub fun main(): Int {
//          let answer = answer()
//          if answer != 42 {
//            panic("?!")
//          }
//          return answer
//        }
//    `)
//
//	runtimeInterface := &testRuntimeInterface{
//		getCode: func(location Location) (bytes []byte, err error) {
//			switch location {
//			case common.StringLocation("imported"):
//				return importedScript, nil
//			default:
//				return nil, fmt.Errorf("unknown import location: %s", location)
//			}
//		},
//	}
//
//	nextTransactionLocation := newTransactionLocationGenerator()
//
//	coverageReport := NewCoverageReport()
//
//	runtime.SetCoverageReport(coverageReport)
//
//	value, err := runtime.ExecuteScript(
//		Script{
//			Source: script,
//		},
//		Context{
//			Interface: runtimeInterface,
//			Location:  nextTransactionLocation(),
//		},
//	)
//	require.NoError(t, err)
//
//	assert.Equal(t, cadence.NewInt(42), value)
//
//	actual, err := json.Marshal(coverageReport)
//	require.NoError(t, err)
//
//	require.JSONEq(t,
//		`
//        {
//          "coverage": {
//            "S.imported": {
//              "line_hits": {
//                "3": 1,
//                "4": 1,
//                "5": 42,
//                "7": 1
//              }
//            },
//            "t.0000000000000000000000000000000000000000000000000000000000000000": {
//              "line_hits": {
//                "5": 1,
//                "6": 1,
//                "9": 1
//              }
//            }
//          }
//        }
//        `,
//		string(actual),
//	)
//}
