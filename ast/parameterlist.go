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
	"sync/atomic"

	"github.com/turbolent/prettier"

	"github.com/onflow/cadence/common"
)

type ParameterList struct {
	parametersByIdentifier atomic.Pointer[map[string]*Parameter]
	Parameters             []*Parameter
	Range
}

func NewParameterList(
	gauge common.MemoryGauge,
	parameters []*Parameter,
	astRange Range,
) *ParameterList {
	common.UseMemory(gauge, common.ParameterListMemoryUsage)
	return &ParameterList{
		Parameters: parameters,
		Range:      astRange,
	}
}

// EffectiveArgumentLabels returns the effective argument labels that
// the arguments of a call must use:
// If no argument label is declared for parameter,
// the parameter name is used as the argument label
func (l *ParameterList) EffectiveArgumentLabels() []string {
	argumentLabels := make([]string, len(l.Parameters))

	for i, parameter := range l.Parameters {
		argumentLabels[i] = parameter.EffectiveArgumentLabel()
	}

	return argumentLabels
}

func (l *ParameterList) ParametersByIdentifier() map[string]*Parameter {
	// Return cached map if already computed
	if cachedMap := l.parametersByIdentifier.Load(); cachedMap != nil {
		return *cachedMap
	}

	// Compute map and cache it
	computedMap := make(map[string]*Parameter, len(l.Parameters))
	for _, parameter := range l.Parameters {
		computedMap[parameter.Identifier.Identifier] = parameter
	}
	l.parametersByIdentifier.Store(&computedMap)
	return computedMap
}

func (l *ParameterList) IsEmpty() bool {
	return l == nil || len(l.Parameters) == 0
}

func (l *ParameterList) Walk(walkChild func(Element)) {
	if l.IsEmpty() {
		return
	}

	for _, parameter := range l.Parameters {
		if parameter.TypeAnnotation != nil {
			walkChild(parameter.TypeAnnotation)
		}
		if parameter.DefaultArgument != nil {
			walkChild(parameter.DefaultArgument)
		}
	}
}

const parameterListEmptyDoc = prettier.Text("()")

var parameterSeparatorDoc prettier.Doc = prettier.Concat{
	prettier.Text(","),
	prettier.Line{},
}

func (l *ParameterList) Doc(ctx PrettyContext) prettier.Doc {

	if len(l.Parameters) == 0 {
		return parameterListEmptyDoc
	}

	// If any parameter carries comments, force a hard-break layout
	// and weave the comments around the commas.
	// Otherwise, use the soft-break layout.
	hardBreak := false
	for _, p := range l.Parameters {
		if p != nil && p.TypeAnnotation != nil && ctx.HasComments(p.TypeAnnotation) {
			hardBreak = true
			break
		}
	}

	if !hardBreak {
		parameterDocs := make([]prettier.Doc, 0, len(l.Parameters))

		for _, parameter := range l.Parameters {
			parameterDocs = append(
				parameterDocs,
				docOrEmpty(parameter, ctx),
			)
		}

		return prettier.WrapParentheses(
			prettier.Join(
				parameterSeparatorDoc,
				parameterDocs...,
			),
			prettier.SoftLine{},
		)
	}

	type paramInfo struct {
		doc      prettier.Doc
		leading  prettier.Doc
		sameLine prettier.Doc
		trailing prettier.Doc
	}

	infos := make([]paramInfo, len(l.Parameters))
	for i, p := range l.Parameters {
		// Take comments BEFORE rendering the parameter,
		// otherwise the parameter's own ctx.Wrap would absorb the same-line comment
		// into its doc and the comma we emit between params would land after it.
		if p != nil && p.TypeAnnotation != nil {
			infos[i].leading, infos[i].sameLine, infos[i].trailing = ctx.Take(p.TypeAnnotation)
		}
		infos[i].doc = docOrEmpty(p, ctx)
	}

	inner := prettier.Concat{}
	for i, info := range infos {
		if i > 0 {
			prev := infos[i-1]
			inner = append(inner, prettier.Text(","))
			if prev.sameLine != nil {
				inner = append(inner, prettier.Text("  "), prev.sameLine)
			}
			inner = append(inner, prettier.HardLine{})
			if prev.trailing != nil {
				inner = append(inner, prev.trailing, prettier.HardLine{})
			}
		}
		if info.leading != nil {
			inner = append(inner, info.leading, prettier.HardLine{})
		}
		inner = append(inner, info.doc)
	}
	last := infos[len(infos)-1]
	if last.sameLine != nil {
		inner = append(inner, prettier.Text("  "), last.sameLine)
	}
	if last.trailing != nil {
		inner = append(inner, prettier.HardLine{}, last.trailing)
	}

	return prettier.Concat{
		prettier.Text("("),
		prettier.Indent{
			Doc: prettier.Concat{
				prettier.HardLine{},
				inner,
			},
		},
		prettier.HardLine{},
		prettier.Text(")"),
	}
}

func (l *ParameterList) String() string {
	return Prettier(l)
}
