/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright Flow Foundation
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

package subtype_gen

import (
	"os"
	"text/template"

	"github.com/dave/dst"
	"github.com/dave/dst/decorator"
	"github.com/dave/dst/decorator/resolver/guess"
)

const headerTemplate = `// Code generated from rules.yaml. DO NOT EDIT.
/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright Flow Foundation
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

`

var parsedHeaderTemplate = template.Must(
	template.New("header").Parse(headerTemplate),
)

func WriteGoFile(outFile *os.File, decls []dst.Decl, packagePath string) {
	err := parsedHeaderTemplate.Execute(outFile, nil)
	if err != nil {
		panic(err)
	}

	resolver := guess.New()
	restorer := decorator.NewRestorerWithImports(packagePath, resolver)

	packageName, err := resolver.ResolvePackage(packagePath)
	if err != nil {
		panic(err)
	}

	err = restorer.Fprint(
		outFile,
		&dst.File{
			Name:  dst.NewIdent(packageName),
			Decls: decls,
		},
	)
	if err != nil {
		panic(err)
	}
}
