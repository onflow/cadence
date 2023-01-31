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
	Prepare        *SpecialFunctionDeclaration
	Roles          []*TransactionRoleDeclaration
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
	roles []*TransactionRoleDeclaration,
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
		Roles:          roles,
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

	for _, role := range d.Roles {
		walkChild(role)
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
		*Alias
		Type string
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

	for _, role := range d.Roles {
		roleDoc := role.Doc()
		addContent(roleDoc)
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
		AsBlockDoc(prettier.Join(
			prettier.HardLine{},
			contents...,
		)),
	)
}

func (d *TransactionDeclaration) String() string {
	return Prettier(d)
}

// TransactionRoleDeclaration

type TransactionRoleDeclaration struct {
	Prepare    *SpecialFunctionDeclaration
	DocString  string
	Fields     []*FieldDeclaration
	Identifier Identifier
	Range
}

var _ Element = &TransactionRoleDeclaration{}
var _ Declaration = &TransactionRoleDeclaration{}
var _ Statement = &TransactionRoleDeclaration{}

func NewTransactionRoleDeclaration(
	gauge common.MemoryGauge,
	identifier Identifier,
	fields []*FieldDeclaration,
	prepare *SpecialFunctionDeclaration,
	docString string,
	declRange Range,
) *TransactionRoleDeclaration {
	common.UseMemory(gauge, common.TransactionRoleDeclarationMemoryUsage)

	return &TransactionRoleDeclaration{
		Identifier: identifier,
		Fields:     fields,
		Prepare:    prepare,
		DocString:  docString,
		Range:      declRange,
	}
}

func (*TransactionRoleDeclaration) ElementType() ElementType {
	return ElementTypeTransactionRoleDeclaration
}

func (d *TransactionRoleDeclaration) Walk(walkChild func(Element)) {
	for _, declaration := range d.Fields {
		walkChild(declaration)
	}
	if d.Prepare != nil {
		walkChild(d.Prepare)
	}

	// TODO: walk pre and post-conditions
}

func (*TransactionRoleDeclaration) isDeclaration() {}
func (*TransactionRoleDeclaration) isStatement()   {}

func (d *TransactionRoleDeclaration) DeclarationIdentifier() *Identifier {
	return &d.Identifier
}

func (d *TransactionRoleDeclaration) DeclarationKind() common.DeclarationKind {
	return common.DeclarationKindTransactionRole
}

func (d *TransactionRoleDeclaration) DeclarationAccess() Access {
	return AccessNotSpecified
}

func (d *TransactionRoleDeclaration) DeclarationMembers() *Members {
	return nil
}

func (d *TransactionRoleDeclaration) DeclarationDocString() string {
	return d.DocString
}

func (d *TransactionRoleDeclaration) MarshalJSON() ([]byte, error) {
	type Alias TransactionRoleDeclaration
	return json.Marshal(&struct {
		*Alias
		Type string
	}{
		Type:  "TransactionRoleDeclaration",
		Alias: (*Alias)(d),
	})
}

var roleKeywordDoc = prettier.Text("role")

func (d *TransactionRoleDeclaration) Doc() prettier.Doc {

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

	return prettier.Concat{
		roleKeywordDoc,
		prettier.Space,
		prettier.Text(d.Identifier.Identifier),
		prettier.Space,
		AsBlockDoc(prettier.Join(
			prettier.HardLine{},
			contents...,
		)),
	}
}

func (d *TransactionRoleDeclaration) String() string {
	return Prettier(d)
}
