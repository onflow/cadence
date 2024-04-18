/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright Dapper Labs, Inc.
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

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	"github.com/kodova/html-to-markdown/escape"
)

var reportPathFlag = flag.String("report", "", "staged contract report JSON file")

func main() {
	flag.Parse()

	reportPath := *reportPathFlag

	content, err := os.ReadFile(reportPath)
	if err != nil {
		panic(err)
	}

	var reportEntries []contractUpdateStatus

	err = json.Unmarshal(content, &reportEntries)
	if err != nil {
		panic(err)
	}

	now := time.Now()

	markdownBuilder := strings.Builder{}
	markdownBuilder.WriteString("## Cadence 1.0 staged contracts migration results\n")
	markdownBuilder.WriteString(fmt.Sprintf("Date: %s\n", now.Format("02 January, 2006")))
	markdownBuilder.WriteString("|Account Address | Contract Name | Status |\n")
	markdownBuilder.WriteString("| --- | --- | --- | \n")

	for _, entry := range reportEntries {
		status := entry.Error
		if status == "" {
			status = "&#9989;"
		} else {
			status = escape.Markdown(status)
			status = strings.ReplaceAll(status, "|", "\\|")
			status = strings.ReplaceAll(status, "\r\n", "<br>")
			status = strings.ReplaceAll(status, "\n", "<br>")
			status = fmt.Sprintf("&#10060;<details><br><summary>Error:</summary><pre>%s</pre></details>", status)
		}
		markdownBuilder.WriteString(
			fmt.Sprintf(
				"| %s | %s | %s | \n",
				entry.AccountAddress,
				entry.ContractName,
				status,
			),
		)
	}

	ext := path.Ext(reportPath)
	mdOutput := fmt.Sprintf("%s.md", reportPath[0:len(reportPath)-len(ext)])

	file, err := os.Create(mdOutput)
	if err != nil {
		panic(err)
	}

	mdContent := markdownBuilder.String()
	_, err = file.Write([]byte(mdContent))
	if err != nil {
		panic(err)
	}

	fmt.Println("Markdown content is written to: ", mdOutput)
}

type contractUpdateStatus struct {
	AccountAddress string `json:"account_address"`
	ContractName   string `json:"contract_name"`
	Error          string `json:"error"`
}
