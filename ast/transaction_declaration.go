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

package ast

import (
	"encoding/json"

	"github.com/turbolent/prettier"

	"github.com/onflow/cadence/common"
)

type TransactionDeclaration struct {
	ParameterList  *ParameterList
	Prepare        *SpecialFunctionDeclaration
	PreConditions  *Conditions
	Execute        *SpecialFunctionDeclaration
	PostConditions *Conditions
	DocString      string
	Fields         []*FieldDeclaration
	Range
}

var _ Element = &TransactionDeclaration{}
var _ Declaration = &TransactionDeclaration{}
var _ Statement = &TransactionDeclaration{}

func NewTransactionDeclaration(
	gauge common.MemoryGauge,
	parameterList *ParameterList,
	fields []*FieldDeclaration,
	prepare *SpecialFunctionDeclaration,
	preConditions *Conditions,
	postConditions *Conditions,
	execute *SpecialFunctionDeclaration,
	docString string,
	declRange Range,
) *TransactionDeclaration {
	common.UseMemory(gauge, common.TransactionDeclarationMemoryUsage)

	return &TransactionDeclaration{
		ParameterList:  parameterList,
		Fields:         fields,
		Prepare:        prepare,
		PreConditions:  preConditions,
		PostConditions: postConditions,
		Execute:        execute,
		DocString:      docString,
		Range:          declRange,
	}
}

func (*TransactionDeclaration) ElementType() ElementType {
	return ElementTypeTransactionDeclaration
}

func (d *TransactionDeclaration) Walk(walkChild func(Element)) {
	if d.ParameterList != nil {
		d.ParameterList.Walk(walkChild)
	}
	for _, declaration := range d.Fields {
		walkChild(declaration)
	}
	if d.PreConditions != nil {
		d.PreConditions.Walk(walkChild)
	}
	if d.Prepare != nil {
		walkChild(d.Prepare)
	}
	if d.PostConditions != nil {
		d.PostConditions.Walk(walkChild)
	}
	if d.Execute != nil {
		walkChild(d.Execute)
	}
}

func (*TransactionDeclaration) isDeclaration() {}
func (*TransactionDeclaration) isStatement()   {}

func (d *TransactionDeclaration) DeclarationIdentifier() *Identifier {
	return nil
}

func (d *TransactionDeclaration) DeclarationKind() common.DeclarationKind {
	return common.DeclarationKindTransaction
}

func (d *TransactionDeclaration) DeclarationAccess() Access {
	return AccessNotSpecified
}

func (d *TransactionDeclaration) DeclarationMembers() *Members {
	return nil
}

func (d *TransactionDeclaration) DeclarationDocString() string {
	return ""
}

func (d *TransactionDeclaration) MarshalJSON() ([]byte, error) {
	type Alias TransactionDeclaration
	return json.Marshal(&struct {
		*Alias
		Type string
	}{
		Type:  "TransactionDeclaration",
		Alias: (*Alias)(d),
	})
}

var transactionKeywordDoc = prettier.Text("transaction")

func (d *TransactionDeclaration) Doc(ctx PrettyContext) prettier.Doc {

	var contents []prettier.Doc

	for _, field := range d.Fields {
		contents = append(contents, field.Doc(ctx))
	}

	if d.Prepare != nil {
		contents = append(contents, d.Prepare.Doc(ctx))
	}

	if conditionsDoc := d.PreConditions.Doc(ctx, preConditionsKeywordDoc); conditionsDoc != nil {
		contents = append(contents, conditionsDoc)
	}

	if d.Execute != nil {
		contents = append(contents, d.Execute.Doc(ctx))
	}

	if conditionsDoc := d.PostConditions.Doc(ctx, postConditionsKeywordDoc); conditionsDoc != nil {
		contents = append(contents, conditionsDoc)
	}

	doc := prettier.Concat{
		transactionKeywordDoc,
	}

	if !d.ParameterList.IsEmpty() {
		doc = append(
			doc,
			d.ParameterList.Doc(ctx),
		)
	}

	body := prettier.Concat{prettier.HardLine{}}
	for i, c := range contents {
		if i > 0 {
			body = append(body, prettier.HardLine{})
		}
		body = append(body, c)
	}

	return ctx.Wrap(d, append(
		doc,
		prettier.Space,
		blockStartDoc,
		prettier.Indent{
			Doc: body,
		},
		prettier.HardLine{},
		blockEndDoc,
	))
}

func (d *TransactionDeclaration) String() string {
	return Prettier(d)
}
