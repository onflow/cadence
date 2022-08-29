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

package ast

import (
	"encoding/json"

	"github.com/turbolent/prettier"

	"github.com/onflow/cadence/runtime/common"
)

type TransactionDeclaration struct {
	ParameterList  *ParameterList
	Fields         []*FieldDeclaration
	Prepare        *SpecialFunctionDeclaration
	PreConditions  *Conditions
	Execute        *SpecialFunctionDeclaration
	PostConditions *Conditions
	DocString      string
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
	// TODO: walk parameters
	for _, declaration := range d.Fields {
		walkChild(declaration)
	}
	if d.Prepare != nil {
		walkChild(d.Prepare)
	}
	if d.Execute != nil {
		walkChild(d.Execute)
	}
	// TODO: walk pre and post-conditions
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
		Type string
		*Alias
	}{
		Type:  "TransactionDeclaration",
		Alias: (*Alias)(d),
	})
}

var transactionKeywordDoc = prettier.Text("transaction")

func (d *TransactionDeclaration) Doc() prettier.Doc {

	var contents []prettier.Doc

	addContent := func(doc prettier.Doc) {
		contents = append(
			contents,
			prettier.Concat{
				prettier.HardLine{},
				doc,
			},
		)
	}

	for _, field := range d.Fields {
		addContent(field.Doc())
	}

	if d.Prepare != nil {
		addContent(d.Prepare.Doc())
	}

	if conditionsDoc := d.PreConditions.Doc(preConditionsKeywordDoc); conditionsDoc != nil {
		addContent(conditionsDoc)
	}

	if d.Execute != nil {
		addContent(d.Execute.Doc())
	}

	if conditionsDoc := d.PostConditions.Doc(postConditionsKeywordDoc); conditionsDoc != nil {
		addContent(conditionsDoc)
	}

	doc := prettier.Concat{
		transactionKeywordDoc,
	}

	if !d.ParameterList.IsEmpty() {
		doc = append(
			doc,
			d.ParameterList.Doc(),
		)
	}

	return append(
		doc,
		prettier.Space,
		blockStartDoc,
		prettier.Indent{
			Doc: prettier.Join(
				prettier.HardLine{},
				contents...,
			),
		},
		prettier.HardLine{},
		blockEndDoc,
	)
}

func (d *TransactionDeclaration) String() string {
	return Prettier(d)
}
