/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2020 Dapper Labs, Inc.
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

package server

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/onflow/cadence/encoding/json"
	"github.com/onflow/cadence/runtime"
	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/parser2"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/stdlib"

	"github.com/onflow/cadence/languageserver/conversion"
	"github.com/onflow/cadence/languageserver/jsonrpc2"
	"github.com/onflow/cadence/languageserver/protocol"
)

var valueDeclarations = append(
	stdlib.FlowBuiltInFunctions(stdlib.FlowBuiltinImpls{}),
	stdlib.BuiltinFunctions...,
).ToSemaValueDeclarations()

var typeDeclarations = append(
	stdlib.FlowBuiltInTypes,
	stdlib.BuiltinTypes...,
).ToTypeDeclarations()

// Document represents an open document on the client. It contains all cached
// information about each document that is used to support CodeLens,
// transaction submission, and script execution.
type Document struct {
	Text    string
	Version float64
}

func (d Document) Offset(line, column int) (offset int) {
	reader := bufio.NewReader(strings.NewReader(d.Text))
	for i := 1; i < line; i++ {
		l, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			return -1
		}
		offset += len(l)
	}
	return offset + column
}

func (d Document) HasAnyPrecedingStringsAtPosition(options []string, line, column int) bool {
	endOffset := d.Offset(line, column)
	if endOffset >= len(d.Text) {
		endOffset = len(d.Text) - 1
	}
	if endOffset < 0 {
		return false
	}

	isWhitespace := func(c byte) bool {
		return c == ' ' || c == '\t' || c == '\n'
	}

	skip := func(predicate func(byte) bool) (done bool) {
		for {
			c := d.Text[endOffset]
			if !predicate(c) {
				break
			}
			endOffset--
			if endOffset < 0 {
				return true
			}
		}

		return false
	}

	// Skip preceding non-whitespace
	done := skip(func(c byte) bool {
		return !isWhitespace(c)
	})
	if done {
		return false
	}

	// Skip preceding whitespace
	done = skip(isWhitespace)
	if done {
		return false
	}

	// Check if any of the options matches

	for _, option := range options {
		optLen := len(option)
		startOffset := endOffset - optLen + 1
		if startOffset < 0 {
			continue
		}

		subStr := d.Text[startOffset : endOffset+1]
		if subStr == option {
			return true
		}
	}

	return false
}

// CommandHandler represents the form of functions that handle commands
// submitted from the client using workspace/executeCommand.
type CommandHandler func(conn protocol.Conn, args ...interface{}) (interface{}, error)

// AddressImportResolver is a function that is used to resolve address imports
//
type AddressImportResolver func(location common.AddressLocation) (string, error)

// AddressContractNamesResolver is a function that is used to resolve contract names of an address
//
type AddressContractNamesResolver func(address common.Address) ([]string, error)

// StringImportResolver is a function that is used to resolve string imports
//
type StringImportResolver func(location common.StringLocation) (string, error)

// CodeLensProvider is a function that is used to provide code lenses for the given checker
//
type CodeLensProvider func(uri protocol.DocumentUri, version float64, checker *sema.Checker) ([]*protocol.CodeLens, error)

// DiagnosticProvider is a function that is used to provide diagnostics for the given checker
//
type DiagnosticProvider func(uri protocol.DocumentUri, version float64, checker *sema.Checker) ([]protocol.Diagnostic, error)

// DocumentSymbolProvider
//
type DocumentSymbolProvider func(uri protocol.DocumentUri, version float64, checker *sema.Checker) ([]*protocol.DocumentSymbol, error)

// InitializationOptionsHandler is a function that is used to handle initialization options sent by the client
//
type InitializationOptionsHandler func(initializationOptions interface{}) error

type Server struct {
	protocolServer       *protocol.Server
	checkers             map[common.LocationID]*sema.Checker
	documents            map[protocol.DocumentUri]Document
	memberResolvers      map[protocol.DocumentUri]map[string]sema.MemberResolver
	ranges               map[protocol.DocumentUri]map[string]sema.Range
	codeActionsResolvers map[protocol.DocumentUri]map[uuid.UUID]func() []*protocol.CodeAction
	// commands is the registry of custom commands we support
	commands map[string]CommandHandler
	// resolveAddressImport is the optional function that is used to resolve address imports
	resolveAddressImport AddressImportResolver
	// resolveAddressContractNames is the optional function that is used to resolve contract names for an address
	resolveAddressContractNames AddressContractNamesResolver
	// resolveStringImport is the optional function that is used to resolve string imports
	resolveStringImport StringImportResolver
	// codeLensProviders are the functions that are used to provide code lenses for a checker
	codeLensProviders []CodeLensProvider
	// diagnosticProviders are the functions that are used to provide diagnostics for a checker
	diagnosticProviders []DiagnosticProvider
	// documentSymbolProviders are the functions that are used to provide information about document symbols for a checker
	documentSymbolProviders []DocumentSymbolProvider
	// initializationOptionsHandlers are the functions that are used to handle initialization options sent by the client
	initializationOptionsHandlers []InitializationOptionsHandler
	accessCheckMode               sema.AccessCheckMode
}

type Option func(*Server) error

type Command struct {
	Name    string
	Handler CommandHandler
}

// WithCommand returns a server options that adds the given command
// to the set of commands provided by the server to the client.
//
// If a command with the given name already exists, the option fails.
//
func WithCommand(command Command) Option {
	return func(s *Server) error {
		if _, ok := s.commands[command.Name]; ok {
			return fmt.Errorf("cannot register command with existing name: %s", command.Name)
		}
		s.commands[command.Name] = command.Handler
		return nil
	}
}

// WithAddressImportResolver returns a server option that sets the given function
// as the function that is used to resolve address imports
//
func WithAddressImportResolver(resolver AddressImportResolver) Option {
	return func(s *Server) error {
		s.resolveAddressImport = resolver
		return nil
	}
}

// WithAddressContractNamesResolver returns a server option that sets the given function
// as the function that is used to resolve contract names of an address
//
func WithAddressContractNamesResolver(resolver AddressContractNamesResolver) Option {
	return func(s *Server) error {
		s.resolveAddressContractNames = resolver
		return nil
	}
}

// WithStringImportResolver returns a server option that sets the given function
// as the function that is used to resolve string imports
//
func WithStringImportResolver(resolver StringImportResolver) Option {
	return func(s *Server) error {
		s.resolveStringImport = resolver
		return nil
	}
}

// WithCodeLensProvider returns a server option that adds the given function
// as a function that is used to generate code lenses
//
func WithCodeLensProvider(provider CodeLensProvider) Option {
	return func(s *Server) error {
		s.codeLensProviders = append(s.codeLensProviders, provider)
		return nil
	}
}

// WithDiagnosticProvider returns a server option that adds the given function
// as a function that is used to generate diagnostics
//
func WithDiagnosticProvider(provider DiagnosticProvider) Option {
	return func(s *Server) error {
		s.diagnosticProviders = append(s.diagnosticProviders, provider)
		return nil
	}
}

// WithInitializationOptionsHandler returns a server option that adds the given function
// as a function that is used to handle initialization options sent by the client
//
func WithInitializationOptionsHandler(handler InitializationOptionsHandler) Option {
	return func(s *Server) error {
		s.initializationOptionsHandlers = append(s.initializationOptionsHandlers, handler)
		return nil
	}
}

const GetEntryPointParametersCommand = "cadence.server.getEntryPointParameters"
const GetContractInitializerParametersCommand = "cadence.server.getContractInitializerParameters"
const ParseEntryPointArgumentsCommand = "cadence.server.parseEntryPointArguments"

func NewServer() (*Server, error) {
	server := &Server{
		checkers:             make(map[common.LocationID]*sema.Checker),
		documents:            make(map[protocol.DocumentUri]Document),
		memberResolvers:      make(map[protocol.DocumentUri]map[string]sema.MemberResolver),
		ranges:               make(map[protocol.DocumentUri]map[string]sema.Range),
		codeActionsResolvers: make(map[protocol.DocumentUri]map[uuid.UUID]func() []*protocol.CodeAction),
		commands:             make(map[string]CommandHandler),
	}
	server.protocolServer = protocol.NewServer(server)

	// Set default commands

	for _, command := range server.defaultCommands() {
		err := server.SetOptions(WithCommand(command))
		if err != nil {
			return nil, err
		}
	}

	return server, nil
}

func (s *Server) SetOptions(options ...Option) error {
	for _, option := range options {
		err := option(s)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Server) Start(stream jsonrpc2.ObjectStream) <-chan struct{} {
	return s.protocolServer.Start(stream)
}

func (s *Server) Stop() error {
	return s.protocolServer.Stop()
}

func (s *Server) checkerForDocument(uri protocol.DocumentUri) *sema.Checker {
	location := uriToLocation(uri)
	return s.checkers[location.ID()]
}

func (s *Server) Initialize(
	conn protocol.Conn,
	params *protocol.InitializeParams,
) (
	*protocol.InitializeResult,
	error,
) {
	result := &protocol.InitializeResult{
		Capabilities: protocol.ServerCapabilities{
			TextDocumentSync:   protocol.Full,
			HoverProvider:      true,
			DefinitionProvider: true,
			CodeLensProvider: &protocol.CodeLensOptions{
				ResolveProvider: false,
			},
			CompletionProvider: &protocol.CompletionOptions{
				TriggerCharacters: []string{"."},
				ResolveProvider:   true,
			},
			DocumentHighlightProvider: true,
			DocumentSymbolProvider:    true,
			RenameProvider:            true,
			SignatureHelpProvider: &protocol.SignatureHelpOptions{
				TriggerCharacters: []string{"("},
			},
			CodeActionProvider: true,
		},
	}

	options := params.InitializationOptions

	s.configure(options)

	for _, handler := range s.initializationOptionsHandlers {
		err := handler(options)
		if err != nil {
			return nil, err
		}
	}

	// after initialization, indicate to the client which commands we support
	go s.registerCommands(conn)

	return result, nil
}

const accessCheckModeOption = "accessCheckMode"

func accessCheckModeFromName(name string) sema.AccessCheckMode {
	switch name {
	case "strict":
		return sema.AccessCheckModeStrict

	case "notSpecifiedRestricted":
		return sema.AccessCheckModeNotSpecifiedRestricted

	case "notSpecifiedUnrestricted":
		return sema.AccessCheckModeNotSpecifiedUnrestricted

	case "none":
		return sema.AccessCheckModeNone

	default:
		return sema.AccessCheckModeStrict
	}
}

func (s *Server) configure(opts interface{}) {
	optsMap, ok := opts.(map[string]interface{})
	if !ok {
		return
	}

	if accessCheckModeName, ok := optsMap[accessCheckModeOption].(string); ok {
		s.accessCheckMode = accessCheckModeFromName(accessCheckModeName)
	} else {
		s.accessCheckMode = sema.AccessCheckModeStrict
	}
}

// Registers the commands that the server is able to handle.
//
// The best reference I've found for how this works is:
// https://stackoverflow.com/questions/43328582/how-to-implement-quickfix-via-a-language-server
func (s *Server) registerCommands(conn protocol.Conn) {

	commandCount := len(s.commands)
	if commandCount <= 0 {
		return
	}

	commands := make([]string, commandCount)
	i := 0
	for name := range s.commands {
		commands[i] = name
		i++
	}

	// Send a message to the client indicating which commands we support
	registration := protocol.RegistrationParams{
		Registrations: []protocol.Registration{
			{
				ID:     "registerCommand",
				Method: "workspace/executeCommand",
				RegisterOptions: protocol.ExecuteCommandRegistrationOptions{
					ExecuteCommandOptions: protocol.ExecuteCommandOptions{
						Commands: commands,
					},
				},
			},
		},
	}

	// We have occasionally observed the client failing to recognize this
	// method if the request is sent too soon after the extension loads.
	// Retrying with a backoff avoids this problem.
	retryAfter := time.Millisecond * 100
	nRetries := 10
	for i = 0; i < nRetries; i++ {
		err := conn.RegisterCapability(&registration)
		if err == nil {
			break
		}
		remainingRetries := nRetries - 1 - i

		conn.LogMessage(&protocol.LogMessageParams{
			Type: protocol.Warning,
			Message: fmt.Sprintf(
				"Failed to register command. Will retry %d more times... err: %s",
				remainingRetries, err.Error(),
			),
		})

		time.Sleep(retryAfter)

		retryAfter *= 2
	}
}

// DidOpenTextDocument is called whenever a new file is opened.
// We parse and check the text and publish diagnostics about the document.
func (s *Server) DidOpenTextDocument(conn protocol.Conn, params *protocol.DidOpenTextDocumentParams) error {

	uri := params.TextDocument.URI
	text := params.TextDocument.Text
	version := params.TextDocument.Version

	s.documents[uri] = Document{
		Text:    text,
		Version: version,
	}

	s.checkAndPublishDiagnostics(conn, uri, text, version)

	return nil
}

// DidChangeTextDocument is called whenever the current document changes.
// We parse and check the text and publish diagnostics about the document.
func (s *Server) DidChangeTextDocument(
	conn protocol.Conn,
	params *protocol.DidChangeTextDocumentParams,
) error {

	uri := params.TextDocument.URI
	text := params.ContentChanges[0].Text
	version := params.TextDocument.Version

	s.documents[uri] = Document{
		Text:    text,
		Version: version,
	}

	s.checkAndPublishDiagnostics(conn, uri, text, version)

	return nil
}

type CadenceCheckCompletedParams struct {

	/*URI defined:
	 * The URI which was checked.
	 */
	URI protocol.DocumentUri `json:"uri"`

	Valid bool `json:"valid"`
}

const cadenceCheckCompletedMethodName = "cadence/checkCompleted"

func (s *Server) checkAndPublishDiagnostics(
	conn protocol.Conn,
	uri protocol.DocumentUri,
	text string,
	version float64,
) {

	diagnostics, _ := s.getDiagnostics(conn, uri, text, version)

	// NOTE: always publish diagnostics and inform the client the checking completed

	_ = conn.PublishDiagnostics(&protocol.PublishDiagnosticsParams{
		URI:         uri,
		Diagnostics: diagnostics,
	})

	valid := true

	for _, diagnostic := range diagnostics {
		if diagnostic.Severity == protocol.SeverityError {
			valid = false
			break
		}
	}

	_ = conn.Notify(cadenceCheckCompletedMethodName, &CadenceCheckCompletedParams{
		URI:   uri,
		Valid: valid,
	})
}

// Hover returns contextual type information about the variable at the given
// location.
func (s *Server) Hover(
	_ protocol.Conn,
	params *protocol.TextDocumentPositionParams,
) (*protocol.Hover, error) {

	uri := params.TextDocument.URI
	checker := s.checkerForDocument(uri)
	if checker == nil {
		return nil, nil
	}

	position := conversion.ProtocolToSemaPosition(params.Position)
	occurrence := checker.Occurrences.Find(position)

	if occurrence == nil || occurrence.Origin == nil {
		return nil, nil
	}

	contents := protocol.MarkupContent{
		Kind:  protocol.Markdown,
		Value: formatType(occurrence.Origin.Type),
	}
	return &protocol.Hover{Contents: contents}, nil
}

func formatType(ty sema.Type) string {
	return fmt.Sprintf("* Type: `%s`", ty.QualifiedString())
}

// Definition finds the definition of the type at the given location.
func (s *Server) Definition(
	_ protocol.Conn,
	params *protocol.TextDocumentPositionParams,
) (
	*protocol.Location,
	error,
) {

	uri := params.TextDocument.URI
	checker := s.checkerForDocument(uri)
	if checker == nil {
		return nil, nil
	}

	position := conversion.ProtocolToSemaPosition(params.Position)
	occurrence := checker.Occurrences.Find(position)

	if occurrence == nil {
		return nil, nil
	}

	origin := occurrence.Origin
	if origin == nil || origin.StartPos == nil || origin.EndPos == nil {
		return nil, nil
	}

	return &protocol.Location{
		URI: uri,
		Range: conversion.ASTToProtocolRange(
			*origin.StartPos,
			*origin.EndPos,
		),
	}, nil
}

func (s *Server) SignatureHelp(
	conn protocol.Conn,
	params *protocol.TextDocumentPositionParams,
) (*protocol.SignatureHelp, error) {

	uri := params.TextDocument.URI
	checker := s.checkerForDocument(uri)
	if checker == nil {
		return nil, nil
	}

	position := conversion.ProtocolToSemaPosition(params.Position)
	invocation := checker.FunctionInvocations.Find(position)

	if invocation == nil {
		return nil, nil
	}

	functionType := invocation.FunctionType

	signatureLabelParts := make([]string, 0, len(functionType.Parameters))

	argumentLabels := functionType.ArgumentLabels()

	for i, parameter := range functionType.Parameters {

		argumentLabel := argumentLabels[i]

		typeAnnotation := parameter.TypeAnnotation.QualifiedString()

		var signatureLabelPart string

		if argumentLabel == sema.ArgumentLabelNotRequired {
			signatureLabelPart = typeAnnotation
		} else {
			signatureLabelPart = fmt.Sprintf(
				"%s: %s",
				argumentLabel,
				typeAnnotation,
			)
		}

		signatureLabelParts = append(signatureLabelParts, signatureLabelPart)
	}

	signatureLabel := fmt.Sprintf(
		"(%s): %s",
		strings.Join(signatureLabelParts, ", "),
		functionType.ReturnTypeAnnotation.QualifiedString(),
	)

	signatureParameters := make([]protocol.ParameterInformation, 0, len(signatureLabelParts))

	for _, part := range signatureLabelParts {
		signatureParameters = append(signatureParameters, protocol.ParameterInformation{
			Label: part,
		})
	}

	var activeParameter int

	for _, trailingSeparatorPosition := range invocation.TrailingSeparatorPositions {
		if position.Compare(sema.ASTToSemaPosition(trailingSeparatorPosition)) > 0 {
			activeParameter++
		}
	}

	return &protocol.SignatureHelp{
		Signatures: []protocol.SignatureInformation{
			{
				Label:      signatureLabel,
				Parameters: signatureParameters,
			},
		},
		ActiveParameter: float64(activeParameter),
	}, nil
}

func (s *Server) DocumentHighlight(
	_ protocol.Conn,
	params *protocol.TextDocumentPositionParams,
) (
	[]*protocol.DocumentHighlight,
	error,
) {
	uri := params.TextDocument.URI
	checker := s.checkerForDocument(uri)
	if checker == nil {
		return nil, nil
	}

	position := conversion.ProtocolToSemaPosition(params.Position)
	occurrences := checker.Occurrences.FindAll(position)
	// If there are no occurrences,
	// then try the preceding position
	if len(occurrences) == 0 && position.Column > 0 {
		previousPosition := position
		previousPosition.Column -= 1
		occurrences = checker.Occurrences.FindAll(previousPosition)
	}

	documentHighlights := make([]*protocol.DocumentHighlight, 0)

	for _, occurrence := range occurrences {

		origin := occurrence.Origin
		if origin == nil || origin.StartPos == nil || origin.EndPos == nil {
			continue
		}

		for _, occurrenceRange := range origin.Occurrences {
			documentHighlights = append(documentHighlights,
				&protocol.DocumentHighlight{
					Range: conversion.ASTToProtocolRange(
						occurrenceRange.StartPos,
						occurrenceRange.EndPos,
					),
				},
			)
		}
	}

	return documentHighlights, nil
}

func (s *Server) Rename(
	_ protocol.Conn,
	params *protocol.RenameParams,
) (
	*protocol.WorkspaceEdit,
	error,
) {
	uri := params.TextDocument.URI
	checker := s.checkerForDocument(uri)
	if checker == nil {
		return nil, nil
	}

	position := conversion.ProtocolToSemaPosition(params.Position)
	occurrences := checker.Occurrences.FindAll(position)
	// If there are no occurrences,
	// then try the preceding position
	if len(occurrences) == 0 && position.Column > 0 {
		previousPosition := position
		previousPosition.Column -= 1
		occurrences = checker.Occurrences.FindAll(previousPosition)
	}

	textEdits := make([]protocol.TextEdit, 0)

	for _, occurrence := range occurrences {

		origin := occurrence.Origin
		if origin == nil || origin.StartPos == nil || origin.EndPos == nil {
			continue
		}

		for _, occurrenceRange := range origin.Occurrences {
			textEdits = append(textEdits,
				protocol.TextEdit{
					Range: conversion.ASTToProtocolRange(
						occurrenceRange.StartPos,
						occurrenceRange.EndPos,
					),
					NewText: params.NewName,
				},
			)
		}
	}

	return &protocol.WorkspaceEdit{
		Changes: &map[string][]protocol.TextEdit{
			string(uri): textEdits,
		},
	}, nil
}

func (s *Server) CodeAction(
	conn protocol.Conn,
	params *protocol.CodeActionParams,
) (
	codeActions []*protocol.CodeAction,
	err error,
) {
	// NOTE: Always initialize to an empty slice, i.e DON'T use nil:
	// The later will be ignored instead of being treated as no items
	codeActions = []*protocol.CodeAction{}

	uri := params.TextDocument.URI
	checker := s.checkerForDocument(uri)
	if checker == nil {
		// Can we ensure this doesn't happen?
		return
	}

	codeActionsResolvers := s.codeActionsResolvers[uri]

	for _, diagnostic := range params.Context.Diagnostics {

		if data, ok := diagnostic.Data.(string); ok {
			codeActionID, err := uuid.Parse(data)
			if err != nil {
				continue
			}

			codeActionsResolver, ok := codeActionsResolvers[codeActionID]
			if !ok {
				continue
			}

			codeActions = append(codeActions,
				codeActionsResolver()...,
			)
		}
	}

	return
}

// CodeLens is called every time the document contents change and returns a
// list of actions to be injected into the source as inline buttons.
func (s *Server) CodeLens(
	_ protocol.Conn,
	params *protocol.CodeLensParams,
) (
	codeLenses []*protocol.CodeLens,
	err error,
) {
	// NOTE: Always initialize to an empty slice, i.e DON'T use nil:
	// The later will be ignored instead of being treated as no items
	codeLenses = []*protocol.CodeLens{}

	uri := params.TextDocument.URI
	checker := s.checkerForDocument(uri)
	if checker == nil {
		// Can we ensure this doesn't happen?
		return
	}

	version := s.documents[uri].Version

	for _, provider := range s.codeLensProviders {
		var moreCodeLenses []*protocol.CodeLens
		moreCodeLenses, err = provider(uri, version, checker)
		if err != nil {
			return
		}

		codeLenses = append(codeLenses, moreCodeLenses...)
	}

	return
}

type CompletionItemData struct {
	URI protocol.DocumentUri `json:"uri"`
	ID  string               `json:"id"`
}

var statementCompletionItems = []*protocol.CompletionItem{
	{
		Kind:             protocol.KeywordCompletion,
		InsertTextFormat: protocol.SnippetTextFormat,
		Label:            "for",
		Detail:           "for-in loop",
		InsertText:       "for $1 in $2 {\n\t$0\n}",
	},
	{
		Kind:             protocol.KeywordCompletion,
		InsertTextFormat: protocol.SnippetTextFormat,
		Label:            "while",
		Detail:           "while loop",
		InsertText:       "while $1 {\n\t$0\n}",
	},
	{
		Kind:             protocol.KeywordCompletion,
		InsertTextFormat: protocol.SnippetTextFormat,
		Label:            "if",
		Detail:           "if statement",
		InsertText:       "if $1 {\n\t$0\n}",
	},
	{
		Kind:             protocol.KeywordCompletion,
		InsertTextFormat: protocol.SnippetTextFormat,
		Label:            "if else",
		Detail:           "if-else statement",
		InsertText:       "if $1 {\n\t$2\n} else {\n\t$0\n}",
	},
	{
		Kind:             protocol.KeywordCompletion,
		InsertTextFormat: protocol.SnippetTextFormat,
		Label:            "else",
		Detail:           "else block",
		InsertText:       "else {\n\t$0\n}",
	},
	{
		Kind:             protocol.KeywordCompletion,
		InsertTextFormat: protocol.SnippetTextFormat,
		Label:            "if let",
		Detail:           "if-let statement",
		InsertText:       "if let $1 = $2 {\n\t$0\n}",
	},
	{
		Kind:             protocol.KeywordCompletion,
		InsertTextFormat: protocol.SnippetTextFormat,
		Label:            "return",
		Detail:           "return statement",
		InsertText:       "return $0",
	},
	{
		Kind:   protocol.KeywordCompletion,
		Label:  "break",
		Detail: "break statement",
	},
	{
		Kind:   protocol.KeywordCompletion,
		Label:  "continue",
		Detail: "continue statement",
	},
	{
		Kind:             protocol.KeywordCompletion,
		InsertTextFormat: protocol.SnippetTextFormat,
		Label:            "emit",
		Detail:           "emit statement",
		InsertText:       "emit $0",
	},
	{
		Kind:             protocol.KeywordCompletion,
		InsertTextFormat: protocol.SnippetTextFormat,
		Label:            "destroy",
		Detail:           "destroy expression",
		InsertText:       "destroy $0",
	},
	{
		Kind:             protocol.KeywordCompletion,
		InsertTextFormat: protocol.SnippetTextFormat,
		Label:            "pre",
		Detail:           "pre conditions",
		InsertText:       "pre {\n\t$0\n}",
	},
	{
		Kind:             protocol.KeywordCompletion,
		InsertTextFormat: protocol.SnippetTextFormat,
		Label:            "post",
		Detail:           "post conditions",
		InsertText:       "post {\n\t$0\n}",
	},
}

var expressionCompletionItems = []*protocol.CompletionItem{
	{
		Kind:             protocol.KeywordCompletion,
		InsertTextFormat: protocol.SnippetTextFormat,
		Label:            "create",
		Detail:           "create statement",
		InsertText:       "create $0",
	},
	{
		Kind:             protocol.KeywordCompletion,
		InsertTextFormat: protocol.SnippetTextFormat,
		Label:            "create",
		Detail:           "create statement",
		InsertText:       "create $0",
	},
	{
		Kind:             protocol.KeywordCompletion,
		InsertTextFormat: protocol.SnippetTextFormat,
		Label:            "let",
		Detail:           "constant declaration",
		InsertText:       "let $1 = $0",
	},
	{
		Kind:             protocol.KeywordCompletion,
		InsertTextFormat: protocol.SnippetTextFormat,
		Label:            "var",
		Detail:           "variable declaration",
		InsertText:       "var $1 = $0",
	},
}

var allAccessOptions = []string{"pub", "priv", "pub(set)", "access(contract)", "access(account)", "access(self)"}
var allAccessOptionsCommaSeparated = strings.Join(allAccessOptions, ",")

var readAccessOptions = []string{"pub", "priv", "access(contract)", "access(account)", "access(self)"}
var readAccessOptionsCommaSeparated = strings.Join(readAccessOptions, ",")

// NOTE: if the document doesn't specify an access modifier yet,
// the completion item's InsertText will  get prefixed with a placeholder
// for the access modifier.
//
// Start placeholders at index 2!
//
var declarationCompletionItems = []*protocol.CompletionItem{
	{
		Kind:             protocol.KeywordCompletion,
		InsertTextFormat: protocol.SnippetTextFormat,
		Label:            "struct",
		Detail:           "struct declaration",
		InsertText:       "struct $2 {\n\t$0\n}",
	},
	{
		Kind:             protocol.KeywordCompletion,
		InsertTextFormat: protocol.SnippetTextFormat,
		Label:            "resource",
		Detail:           "resource declaration",
		InsertText:       "resource $2 {\n\t$0\n}",
	},
	{
		Kind:             protocol.KeywordCompletion,
		InsertTextFormat: protocol.SnippetTextFormat,
		Label:            "contract",
		Detail:           "contract declaration",
		InsertText:       "contract $2 {\n\t$0\n}",
	},
	{
		Kind:             protocol.KeywordCompletion,
		InsertTextFormat: protocol.SnippetTextFormat,
		Label:            "struct interface",
		Detail:           "struct interface declaration",
		InsertText:       "struct interface $2 {\n\t$0\n}",
	},
	{
		Kind:             protocol.KeywordCompletion,
		InsertTextFormat: protocol.SnippetTextFormat,
		Label:            "resource interface",
		Detail:           "resource interface declaration",
		InsertText:       "resource interface $2 {\n\t$0\n}",
	},
	{
		Kind:             protocol.KeywordCompletion,
		InsertTextFormat: protocol.SnippetTextFormat,
		Label:            "contract interface",
		Detail:           "contract interface declaration",
		InsertText:       "contract interface $2 {\n\t$0\n}",
	},
	{
		Kind:             protocol.KeywordCompletion,
		InsertTextFormat: protocol.SnippetTextFormat,
		Label:            "event",
		Detail:           "event declaration",
		InsertText:       "event $2($0)",
	},
	{
		Kind:             protocol.KeywordCompletion,
		InsertTextFormat: protocol.SnippetTextFormat,
		Label:            "fun",
		Detail:           "function declaration",
		InsertText:       "fun $2($3)${4:: $5} {\n\t$0\n}",
	},
	{
		Kind:             protocol.KeywordCompletion,
		InsertTextFormat: protocol.SnippetTextFormat,
		Label:            "enum",
		Detail:           "enum declaration",
		InsertText:       "enum $2: $3 {\n\t$0\n}",
	},
}

// NOTE: if the document doesn't specify an access modifier yet,
// the completion item's InsertText will  get prefixed with a placeholder
// for the access modifier.
//
// Start placeholders at index 2!
//
var containerCompletionItems = []*protocol.CompletionItem{
	{
		Kind:             protocol.KeywordCompletion,
		InsertTextFormat: protocol.SnippetTextFormat,
		Label:            "var",
		Detail:           "variable field",
		InsertText:       "var $2: $0",
	},
	{
		Kind:             protocol.KeywordCompletion,
		InsertTextFormat: protocol.SnippetTextFormat,
		Label:            "let",
		Detail:           "constant field",
		InsertText:       "let $2: $0",
	},
	// alias for the above
	{
		Kind:             protocol.KeywordCompletion,
		InsertTextFormat: protocol.SnippetTextFormat,
		Label:            "const",
		Detail:           "constant field",
		InsertText:       "let $2: $0",
	},
}

// Completion is called to compute completion items at a given cursor position.
//
func (s *Server) Completion(
	conn protocol.Conn,
	params *protocol.CompletionParams,
) (
	items []*protocol.CompletionItem,
	err error,
) {
	// NOTE: Always initialize to an empty slice, i.e DON'T use nil:
	// The later will be ignored instead of being treated as no items
	items = []*protocol.CompletionItem{}

	uri := params.TextDocument.URI
	checker := s.checkerForDocument(uri)
	if checker == nil {
		return
	}

	document, ok := s.documents[uri]
	if !ok {
		return
	}

	position := conversion.ProtocolToSemaPosition(params.Position)

	memberCompletions := s.memberCompletions(position, checker, uri)
	if len(memberCompletions) > 0 {
		return memberCompletions, nil
	}

	// prioritize range completion items over other items
	rangeCompletions := s.rangeCompletions(position, checker, uri)
	for _, item := range rangeCompletions {
		item.SortText = fmt.Sprintf("1" + item.Label)
	}
	items = append(items, rangeCompletions...)

	// TODO: make conditional on position being inside a function declaration
	items = append(items, statementCompletionItems...)

	// TODO: make conditional on position being inside a function declaration
	items = append(items, expressionCompletionItems...)

	requiresAccessModifierPlaceholder :=
		!document.HasAnyPrecedingStringsAtPosition(allAccessOptions, position.Line, position.Column)

	// TODO: make conditional on position being outside a function declaration
	for _, item := range declarationCompletionItems {
		if requiresAccessModifierPlaceholder {
			item = withCompletionItemInsertText(
				item,
				fmt.Sprintf("${1|%s|} %s", readAccessOptionsCommaSeparated, item.InsertText),
			)
		}
		items = append(items, item)
	}

	// TODO: make conditional on position being inside a container, but not a function declaration
	for _, item := range containerCompletionItems {
		if requiresAccessModifierPlaceholder {
			item = withCompletionItemInsertText(
				item,
				fmt.Sprintf("${1|%s|} %s", allAccessOptionsCommaSeparated, item.InsertText),
			)
		}
		items = append(items, item)
	}

	return
}

func withCompletionItemInsertText(item *protocol.CompletionItem, insertText string) *protocol.CompletionItem {
	itemCopy := *item
	itemCopy.InsertText = insertText
	return &itemCopy
}

func (s *Server) memberCompletions(
	position sema.Position,
	checker *sema.Checker,
	uri protocol.DocumentUri,
) (items []*protocol.CompletionItem) {

	// The client asks for the column after the identifier,
	// query the member accesses for the preceding position

	if position.Column > 0 {
		position.Column -= 1
	}
	memberAccess := checker.MemberAccesses.Find(position)

	delete(s.memberResolvers, uri)

	if memberAccess == nil {
		return
	}

	memberResolvers := memberAccess.AccessedType.GetMembers()
	s.memberResolvers[uri] = memberResolvers

	for name, resolver := range memberResolvers {
		kind := convertDeclarationKindToCompletionItemType(resolver.Kind)
		commitCharacters := declarationKindCommitCharacters(resolver.Kind)

		item := &protocol.CompletionItem{
			Label:            name,
			Kind:             kind,
			CommitCharacters: commitCharacters,
			Data: CompletionItemData{
				URI: uri,
				ID:  name,
			},
		}

		// If the member is a function, also prepare the argument list
		// with placeholders and suggest it

		if resolver.Kind == common.DeclarationKindFunction {
			s.prepareFunctionMemberCompletionItem(item, resolver, name)

			// If we are completing a function, we should also trigger signature help
			item.Command = &protocol.Command{
				Command: "editor.action.triggerParameterHints",
			}
		}

		items = append(items, item)
	}

	return items
}

func (s *Server) rangeCompletions(
	position sema.Position,
	checker *sema.Checker,
	uri protocol.DocumentUri,
) (items []*protocol.CompletionItem) {

	ranges := checker.Ranges.FindAll(position)

	delete(s.ranges, uri)

	if ranges == nil {
		return
	}

	resolvers := make(map[string]sema.Range, len(ranges))
	s.ranges[uri] = resolvers

	for index, r := range ranges {
		id := strconv.Itoa(index)
		kind := convertDeclarationKindToCompletionItemType(r.DeclarationKind)
		item := &protocol.CompletionItem{
			Label: r.Identifier,
			Kind:  kind,
			Data: CompletionItemData{
				URI: uri,
				ID:  id,
			},
		}

		resolvers[id] = r

		// If the range is for a function, also prepare the argument list
		// with placeholders and suggest it

		var isFunctionCompletion bool

		switch r.DeclarationKind {
		case common.DeclarationKindFunction:
			functionType := r.Type.(*sema.FunctionType)
			s.prepareParametersCompletionItem(
				item,
				r.Identifier,
				functionType.Parameters,
			)

			isFunctionCompletion = true

		case common.DeclarationKindStructure,
			common.DeclarationKindResource,
			common.DeclarationKindEvent:

			if constructorFunctionType, ok := r.Type.(*sema.ConstructorFunctionType); ok {
				item.Kind = protocol.ConstructorCompletion

				s.prepareParametersCompletionItem(
					item,
					r.Identifier,
					constructorFunctionType.Parameters,
				)

				isFunctionCompletion = true
			}
		}

		// If we are completing a function, we should also trigger signature help
		if isFunctionCompletion {
			item.Command = &protocol.Command{
				Command: "editor.action.triggerParameterHints",
			}
		}

		items = append(items, item)
	}

	return items
}

func (s *Server) prepareFunctionMemberCompletionItem(
	item *protocol.CompletionItem,
	resolver sema.MemberResolver,
	name string,
) {
	member := resolver.Resolve(item.Label, ast.Range{}, func(err error) { /* NO-OP */ })
	functionType, ok := member.TypeAnnotation.Type.(*sema.FunctionType)
	if !ok {
		return
	}

	s.prepareParametersCompletionItem(item, name, functionType.Parameters)
}

func (s *Server) prepareParametersCompletionItem(
	item *protocol.CompletionItem,
	name string,
	parameters []*sema.Parameter,
) {
	item.InsertTextFormat = protocol.SnippetTextFormat

	var builder strings.Builder
	builder.WriteString(name)
	builder.WriteRune('(')

	for i, parameter := range parameters {
		if i > 0 {
			builder.WriteString(", ")
		}
		label := parameter.EffectiveArgumentLabel()
		if label != sema.ArgumentLabelNotRequired {
			builder.WriteString(label)
			builder.WriteString(": ")
		}
		builder.WriteString("${")
		builder.WriteString(strconv.Itoa(i + 1))
		builder.WriteRune(':')
		builder.WriteString(parameter.Identifier)
		builder.WriteRune('}')
	}

	builder.WriteRune(')')
	item.InsertText = builder.String()
}

func convertDeclarationKindToCompletionItemType(kind common.DeclarationKind) protocol.CompletionItemKind {
	switch kind {
	case common.DeclarationKindFunction:
		return protocol.FunctionCompletion

	case common.DeclarationKindField:
		return protocol.FieldCompletion

	case common.DeclarationKindStructure,
		common.DeclarationKindResource,
		common.DeclarationKindEvent,
		common.DeclarationKindContract,
		common.DeclarationKindType:
		return protocol.ClassCompletion

	case common.DeclarationKindStructureInterface,
		common.DeclarationKindResourceInterface,
		common.DeclarationKindContractInterface:
		return protocol.InterfaceCompletion

	case common.DeclarationKindVariable:
		return protocol.VariableCompletion

	case common.DeclarationKindConstant,
		common.DeclarationKindParameter:
		return protocol.ConstantCompletion

	default:
		return protocol.TextCompletion
	}
}

func declarationKindCommitCharacters(kind common.DeclarationKind) []string {
	switch kind {
	case common.DeclarationKindField:
		return []string{"."}

	default:
		return nil
	}
}

// ResolveCompletionItem is called to compute completion items at a given cursor position.
//
func (s *Server) ResolveCompletionItem(
	_ protocol.Conn,
	item *protocol.CompletionItem,
) (
	result *protocol.CompletionItem,
	err error,
) {
	result = item

	var data CompletionItemData
	cfg := &mapstructure.DecoderConfig{
		Metadata: nil,
		Result:   &data,
		TagName:  "json",
	}
	decoder, _ := mapstructure.NewDecoder(cfg)
	err = decoder.Decode(item.Data)
	if err != nil {
		return
	}

	resolved := s.maybeResolveMember(data.URI, data.ID, result)
	if resolved {
		return
	}

	resolved = s.maybeResolveRange(data.URI, data.ID, result)
	if resolved {
		return
	}

	return
}

func (s *Server) maybeResolveMember(uri protocol.DocumentUri, id string, result *protocol.CompletionItem) bool {
	memberResolvers, ok := s.memberResolvers[uri]
	if !ok {
		return false
	}

	resolver, ok := memberResolvers[id]
	if !ok {
		return false
	}

	member := resolver.Resolve(result.Label, ast.Range{}, func(err error) { /* NO-OP */ })

	result.Documentation = protocol.MarkupContent{
		Kind:  "markdown",
		Value: member.DocString,
	}

	switch member.DeclarationKind {
	case common.DeclarationKindField:
		typeString := member.TypeAnnotation.Type.QualifiedString()

		result.Detail = fmt.Sprintf(
			"%s.%s: %s",
			member.ContainerType.String(),
			member.Identifier,
			typeString,
		)

		// add the variable kind, if any, as a prefix
		if member.VariableKind != ast.VariableKindNotSpecified {
			result.Detail = fmt.Sprintf("(%s) %s",
				member.VariableKind.Name(),
				result.Detail,
			)
		}

	case common.DeclarationKindFunction:
		typeString := member.TypeAnnotation.Type.QualifiedString()

		result.Detail = fmt.Sprintf(
			"(function) %s.%s%s",
			member.ContainerType.String(),
			member.Identifier,
			typeString[1:len(typeString)-1],
		)

	case common.DeclarationKindStructure,
		common.DeclarationKindResource,
		common.DeclarationKindEvent,
		common.DeclarationKindContract,
		common.DeclarationKindStructureInterface,
		common.DeclarationKindResourceInterface,
		common.DeclarationKindContractInterface:

		result.Detail = fmt.Sprintf(
			"(%s) %s.%s",
			member.DeclarationKind.Name(),
			member.ContainerType.String(),
			member.Identifier,
		)
	}

	return true
}

func (s *Server) maybeResolveRange(uri protocol.DocumentUri, id string, result *protocol.CompletionItem) bool {
	ranges, ok := s.ranges[uri]
	if !ok {
		return false
	}

	r, ok := ranges[id]
	if !ok {
		return false
	}

	if constructorFunctionType, ok := r.Type.(*sema.ConstructorFunctionType); ok {
		typeString := constructorFunctionType.QualifiedString()

		result.Detail = fmt.Sprintf(
			"(constructor) %s",
			typeString[1:len(typeString)-1],
		)

	} else {
		result.Detail = fmt.Sprintf(
			"(%s) %s",
			r.DeclarationKind.Name(),
			r.Type.String(),
		)
	}

	result.Documentation = r.DocString

	return true
}

// ExecuteCommand is called to execute a custom, server-defined command.
//
// We register all the commands we support in registerCommands and populate
// their corresponding handler at server initialization.
func (s *Server) ExecuteCommand(conn protocol.Conn, params *protocol.ExecuteCommandParams) (interface{}, error) {

	conn.LogMessage(&protocol.LogMessageParams{
		Type:    protocol.Log,
		Message: fmt.Sprintf("called execute command: %s", params.Command),
	})

	f, ok := s.commands[params.Command]
	if !ok {
		return nil, fmt.Errorf("invalid command: %s", params.Command)
	}
	return f(conn, params.Arguments...)
}

// DocumentSymbol is called every time the document contents change and returns a
// tree of known  document symbols, which can be shown in outline panel
func (s *Server) DocumentSymbol(
	_ protocol.Conn,
	params *protocol.DocumentSymbolParams,
) (
	symbols []*protocol.DocumentSymbol,
	err error,
) {

	// NOTE: Always initialize to an empty slice, i.e DON'T use nil:
	// The later will be ignored instead of being treated as no items
	symbols = []*protocol.DocumentSymbol{}

	// get uri from parameters caught by grpc server
	uri := params.TextDocument.URI
	checker := s.checkerForDocument(uri)
	if checker == nil {
		return
	}

	for _, declaration := range checker.Program.Declarations() {
		symbol := conversion.DeclarationToDocumentSymbol(declaration)
		symbols = append(symbols, &symbol)
	}

	return
}

// Shutdown tells the server to stop accepting any new requests. This can only
// be followed by a call to Exit, which exits the process.
func (*Server) Shutdown(conn protocol.Conn) error {

	conn.ShowMessage(&protocol.ShowMessageParams{
		Type:    protocol.Warning,
		Message: "Cadence language server is shutting down",
	})

	return nil
}

// Exit exits the process.
func (*Server) Exit(_ protocol.Conn) error {
	os.Exit(0)
	return nil
}

const filePrefix = "file://"

// getDiagnostics parses and checks the given file and generates diagnostics
// indicating each syntax or semantic error. Returns a list of diagnostics
// that the caller is responsible for publishing to the client.
//
// Returns an error if an unexpected error occurred.
func (s *Server) getDiagnostics(
	conn protocol.Conn,
	uri protocol.DocumentUri,
	text string,
	version float64,
) (
	diagnostics []protocol.Diagnostic,
	diagnosticsErr error,
) {
	// Always reset the code actions for this document
	codeActionsResolvers := map[uuid.UUID]func() []*protocol.CodeAction{}
	s.codeActionsResolvers[uri] = codeActionsResolvers

	// NOTE: Always initialize to an empty slice, i.e DON'T use nil:
	// The later will be ignored instead of being treated as no items
	diagnostics = []protocol.Diagnostic{}

	program, parseError := parse(conn, text, string(uri))

	// If there were parsing errors, convert each one to a diagnostic and exit
	// without checking.

	if parseError != nil {
		if parentErr, ok := parseError.(errors.ParentError); ok {
			parserDiagnostics := s.getDiagnosticsForParentError(conn, uri, parentErr, codeActionsResolvers)
			diagnostics = append(diagnostics, parserDiagnostics...)
		}
	}

	// If there is a parse result succeeded proceed with resolving imports and checking the parsed program,
	// even if there there might have been parsing errors.

	location := uriToLocation(uri)

	if program == nil {
		delete(s.checkers, location.ID())
		return
	}

	var checker *sema.Checker
	checker, diagnosticsErr = sema.NewChecker(
		program,
		location,
		sema.WithPredeclaredValues(valueDeclarations),
		sema.WithPredeclaredTypes(typeDeclarations),
		sema.WithLocationHandler(
			func(identifiers []ast.Identifier, location common.Location) ([]sema.ResolvedLocation, error) {
				addressLocation, isAddress := location.(common.AddressLocation)

				// if the location is not an address location, e.g. an identifier location (`import Crypto`),
				// then return a single resolved location which declares all identifiers.

				if !isAddress {
					return []runtime.ResolvedLocation{
						{
							Location:    location,
							Identifiers: identifiers,
						},
					}, nil
				}

				// if the location is an address,
				// and no specific identifiers where requested in the import statement,
				// then fetch all identifiers at this address

				if len(identifiers) == 0 {
					// if there is no contract name resolver,
					// then return no resolved locations

					if s.resolveAddressContractNames == nil {
						return nil, nil
					}

					contractNames, err := s.resolveAddressContractNames(addressLocation.Address)
					if err != nil {
						panic(err)
					}

					// if there are no contracts deployed,
					// then return no resolved locations

					if len(contractNames) == 0 {
						return nil, nil
					}

					identifiers = make([]ast.Identifier, len(contractNames))

					for i := range identifiers {
						identifiers[i] = runtime.Identifier{
							Identifier: contractNames[i],
						}
					}
				}

				// return one resolved location per identifier.
				// each resolved location is an address contract location

				resolvedLocations := make([]runtime.ResolvedLocation, len(identifiers))
				for i := range resolvedLocations {
					identifier := identifiers[i]
					resolvedLocations[i] = runtime.ResolvedLocation{
						Location: common.AddressLocation{
							Address: addressLocation.Address,
							Name:    identifier.Identifier,
						},
						Identifiers: []runtime.Identifier{identifier},
					}
				}

				return resolvedLocations, nil
			},
		),
		sema.WithPositionInfoEnabled(true),
		sema.WithImportHandler(
			func(checker *sema.Checker, importedLocation common.Location, importRange ast.Range) (sema.Import, error) {
				switch importedLocation {
				case stdlib.CryptoChecker.Location:
					return sema.ElaborationImport{
						Elaboration: stdlib.CryptoChecker.Elaboration,
					}, nil

				default:
					if isPathLocation(importedLocation) {
						// import may be a relative path and therefore should be normalized
						// against the current location
						importedLocation = normalizePathLocation(checker.Location, importedLocation)

						if checker.Location == importedLocation {
							return nil, &sema.CheckerError{
								Errors: []error{fmt.Errorf("cannot import current file: %s", importedLocation)},
							}
						}
					}

					importedLocationID := importedLocation.ID()

					importedChecker, ok := s.checkers[importedLocationID]
					if !ok {
						importedProgram, err := s.resolveImport(importedLocation)
						if err != nil {
							return nil, err
						}

						importedChecker, err = checker.SubChecker(importedProgram, importedLocation)
						if err != nil {
							return nil, err
						}
						s.checkers[importedLocationID] = importedChecker
						err = importedChecker.Check()
						if err != nil {
							return nil, err
						}
					}

					return sema.ElaborationImport{
						Elaboration: importedChecker.Elaboration,
					}, nil
				}
			},
		),
		sema.WithAccessCheckMode(s.accessCheckMode),
	)
	if diagnosticsErr != nil {
		return
	}

	start := time.Now()
	checkError := checker.Check()
	elapsed := time.Since(start)

	// Log how long it took to check the file
	conn.LogMessage(&protocol.LogMessageParams{
		Type:    protocol.Info,
		Message: fmt.Sprintf("checking %s took %s", string(uri), elapsed),
	})

	s.checkers[location.ID()] = checker

	if checkError != nil {
		if parentErr, ok := checkError.(errors.ParentError); ok {
			checkerDiagnostics := s.getDiagnosticsForParentError(conn, uri, parentErr, codeActionsResolvers)
			diagnostics = append(diagnostics, checkerDiagnostics...)
		}
	}

	for _, provider := range s.diagnosticProviders {
		var extraDiagnostics []protocol.Diagnostic
		extraDiagnostics, diagnosticsErr = provider(uri, version, checker)
		if diagnosticsErr != nil {
			return
		}
		diagnostics = append(diagnostics, extraDiagnostics...)
	}

	for _, hint := range checker.Hints() {
		diagnostic, codeActionsResolver := convertHint(hint, uri)
		if codeActionsResolver != nil {
			codeActionsResolverID := uuid.New()
			diagnostic.Data = codeActionsResolverID
			codeActionsResolvers[codeActionsResolverID] = codeActionsResolver
		}
		diagnostics = append(diagnostics, diagnostic)
	}

	return
}

// getDiagnosticsForParentError unpacks all child errors and converts each to
// a diagnostic. Both parser and checker errors can be unpacked.
//
// Logs any conversion failures to the client.
func (s *Server) getDiagnosticsForParentError(
	conn protocol.Conn,
	uri protocol.DocumentUri,
	err errors.ParentError,
	codeActionsResolvers map[uuid.UUID]func() []*protocol.CodeAction,
) (
	diagnostics []protocol.Diagnostic,
) {
	for _, childErr := range err.ChildErrors() {
		convertibleErr, ok := childErr.(convertibleError)
		if !ok {
			conn.LogMessage(&protocol.LogMessageParams{
				Type:    protocol.Error,
				Message: fmt.Sprintf("Unable to convert non-convertable error to diagnostic: %T", childErr),
			})
			continue
		}
		diagnostic, codeActionsResolver := s.convertError(convertibleErr, uri)
		if codeActionsResolver != nil {
			codeActionsResolverID := uuid.New()
			diagnostic.Data = codeActionsResolverID
			codeActionsResolvers[codeActionsResolverID] = codeActionsResolver
		}

		if errorNotes, ok := convertibleErr.(errors.ErrorNotes); ok {
			for _, errorNote := range errorNotes.ErrorNotes() {
				if positioned, hasPosition := errorNote.(ast.HasPosition); hasPosition {
					startPos := positioned.StartPosition()
					endPos := positioned.EndPosition()
					message := errorNote.Message()

					diagnostic.RelatedInformation = append(diagnostic.RelatedInformation,
						protocol.DiagnosticRelatedInformation{
							Location: protocol.Location{
								URI:   uri,
								Range: conversion.ASTToProtocolRange(startPos, endPos),
							},
							Message: message,
						},
					)
				}
			}
		}

		diagnostics = append(diagnostics, diagnostic)
	}

	return
}

// parse parses the given code and returns the resultant program.
func parse(conn protocol.Conn, code, location string) (*ast.Program, error) {
	start := time.Now()
	program, err := parser2.ParseProgram(code)
	elapsed := time.Since(start)

	conn.LogMessage(&protocol.LogMessageParams{
		Type:    protocol.Info,
		Message: fmt.Sprintf("parsing %s took %s", location, elapsed),
	})

	return program, err
}

func (s *Server) resolveImport(location common.Location) (program *ast.Program, err error) {
	// NOTE: important, *DON'T* return an error when a location type
	// is not supported: the import location can simply not be resolved,
	// no error occurred while resolving it.
	//
	// For example, the Crypto contract has an IdentifierLocation,
	// and we simply return no code for it, so that the checker's
	// import handler is called which resolves the location

	var code string
	switch loc := location.(type) {
	case common.StringLocation:
		if s.resolveStringImport == nil {
			return nil, nil
		}

		code, err = s.resolveStringImport(loc)

	case common.AddressLocation:
		if s.resolveAddressImport == nil {
			return nil, nil
		}
		code, err = s.resolveAddressImport(loc)

	default:
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return parser2.ParseProgram(code)
}

func (s *Server) GetDocument(uri protocol.DocumentUri) (doc Document, ok bool) {
	doc, ok = s.documents[uri]
	return
}

func (s *Server) defaultCommands() []Command {
	return []Command{
		{
			Name:    GetEntryPointParametersCommand,
			Handler: s.getEntryPointParameters,
		},
		{
			Name:    GetContractInitializerParametersCommand,
			Handler: s.getContractInitializerParameters,
		},
		{
			Name:    ParseEntryPointArgumentsCommand,
			Handler: s.parseEntryPointArguments,
		},
	}
}

// getEntryPointParameters returns the script or transaction parameters of the source document.
//
// There should be exactly 1 argument:
//   * the DocumentURI of the file to submit
func (s *Server) getEntryPointParameters(_ protocol.Conn, args ...interface{}) (interface{}, error) {

	err := CheckCommandArgumentCount(args, 1)
	if err != nil {
		return nil, err
	}

	uriArg, ok := args[0].(string)
	if !ok {
		return nil, fmt.Errorf("invalid URI argument: %#+v", args[0])
	}

	uri := protocol.DocumentUri(uriArg)
	checker := s.checkerForDocument(uri)
	if checker == nil {
		return nil, fmt.Errorf("could not find document for URI %s", uri)
	}

	parameters := checker.EntryPointParameters()

	encodedParameters := encodeParameters(parameters)

	return encodedParameters, nil
}

// getContractInitializerParameters returns the parameters of the sole contract's initializer in the source document,
// or none if no initializer is declared, or the program contains none or more than one contract declaration.
//
// There should be exactly 1 argument:
//   * the DocumentURI of the file to submit
func (s *Server) getContractInitializerParameters(_ protocol.Conn, args ...interface{}) (interface{}, error) {

	err := CheckCommandArgumentCount(args, 1)
	if err != nil {
		return nil, err
	}

	uriArg, ok := args[0].(string)
	if !ok {
		return nil, fmt.Errorf("invalid URI argument: %#+v", args[0])
	}

	uri := protocol.DocumentUri(uriArg)
	checker := s.checkerForDocument(uri)
	if checker == nil {
		return nil, fmt.Errorf("could not find document for URI %s", uri)
	}

	compositeDeclarations := checker.Program.CompositeDeclarations()
	if len(compositeDeclarations) != 1 {
		// NOTE: return allocated slice, so result is `[]` in JSON, nil is serialized to `null`
		return []Parameter{}, nil
	}

	compositeDeclaration := compositeDeclarations[0]
	if compositeDeclaration.CompositeKind != common.CompositeKindContract {
		// NOTE: return allocated slice, so result is `[]` in JSON, nil is serialized to `null`
		return []Parameter{}, nil
	}

	compositeType := checker.Elaboration.CompositeDeclarationTypes[compositeDeclaration]

	encodedParameters := encodeParameters(compositeType.ConstructorParameters)

	return encodedParameters, nil
}

// parseEntryPointArguments returns the values for the given arguments (literals) for the entry point.
//
// There should be exactly 2 arguments:
//   * the DocumentURI of the file to submit
//   * the array of arguments
func (s *Server) parseEntryPointArguments(_ protocol.Conn, args ...interface{}) (interface{}, error) {

	err := CheckCommandArgumentCount(args, 2)
	if err != nil {
		return nil, err
	}

	uriArg, ok := args[0].(string)
	if !ok {
		return nil, fmt.Errorf("invalid URI argument: %#+v", args[0])
	}

	uri := protocol.DocumentUri(uriArg)
	checker := s.checkerForDocument(uri)
	if checker == nil {
		return nil, fmt.Errorf("could not find document for URI %s", uri)
	}

	arguments, ok := args[1].([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid arguments argument: %#+v", args[1])
	}

	parameters := checker.EntryPointParameters()

	argumentCount := len(arguments)
	parameterCount := len(parameters)
	if argumentCount != parameterCount {
		return nil, fmt.Errorf(
			"invalid number of arguments: got %d, expected %d",
			argumentCount,
			parameterCount,
		)
	}

	result := make([]interface{}, len(arguments))

	for i, argument := range arguments {
		parameter := parameters[i]
		parameterType := parameter.TypeAnnotation.Type

		argumentCode, ok := argument.(string)
		if !ok {
			return nil, fmt.Errorf("invalid argument at index %d: %#+v", i, argument)
		}
		value, err := runtime.ParseLiteral(argumentCode, parameterType)
		if err != nil {
			return nil, err
		}

		result[i] = json.Prepare(value)
	}

	return result, nil
}

// convertibleError is an error that can be converted to LSP diagnostic.
type convertibleError interface {
	error
	ast.HasPosition
}

// convertError converts a checker error to a diagnostic
// and an optional code action to resolve the error.
//
func (s *Server) convertError(
	err convertibleError,
	uri protocol.DocumentUri,
) (
	protocol.Diagnostic,
	func() []*protocol.CodeAction,
) {
	startPosition := err.StartPosition()
	endPosition := err.EndPosition()

	protocolRange := conversion.ASTToProtocolRange(startPosition, endPosition)

	var message strings.Builder
	message.WriteString(err.Error())

	if secondaryError, ok := err.(errors.SecondaryError); ok {
		message.WriteString(". ")
		message.WriteString(secondaryError.SecondaryError())
	}

	diagnostic := protocol.Diagnostic{
		Message:  message.String(),
		Severity: protocol.SeverityError,
		Range:    protocolRange,
	}

	var codeActionsResolver func() []*protocol.CodeAction

	switch err := err.(type) {
	case *sema.TypeMismatchError:
		codeActionsResolver = s.maybeReturnTypeChangeCodeActionsResolver(diagnostic, err, uri)

	case *sema.ConformanceError:
		codeActionsResolver = maybeAddMissingMembersCodeActionResolver(diagnostic, err, uri)

	case *sema.NotDeclaredError:
		codeActionsResolver = s.maybeAddVariableDeclarationActionsResolver(diagnostic, err, uri)
	}

	return diagnostic, codeActionsResolver
}

func (s *Server) maybeReturnTypeChangeCodeActionsResolver(
	diagnostic protocol.Diagnostic,
	err *sema.TypeMismatchError,
	uri protocol.DocumentUri,
) func() []*protocol.CodeAction {

	// The type mismatch could be in a return statement
	// due to a missing or wrong return type.
	//
	// Find the expression,
	// its parent return statement,
	// its parent function expression or function declaration,
	// and suggest adding the return type.

	if err.Expression == nil {
		return nil
	}

	checker := s.checkerForDocument(uri)
	if checker == nil {
		return nil
	}

	var foundReturn bool
	var parameterList *ast.ParameterList
	var returnTypeAnnotation *ast.TypeAnnotation

	var stack []ast.Element
	ast.Inspect(checker.Program, func(element ast.Element) bool {

		switch element {
		case err.Expression:
			for i := len(stack) - 1; i >= 0; i-- {
				parent := stack[i]
				switch parent := parent.(type) {
				case *ast.ReturnStatement:
					if parent.Expression == err.Expression {
						foundReturn = true
					}

				case *ast.FunctionDeclaration:
					parameterList = parent.ParameterList
					returnTypeAnnotation = parent.ReturnTypeAnnotation

				case *ast.FunctionExpression:
					parameterList = parent.ParameterList
					returnTypeAnnotation = parent.ReturnTypeAnnotation
				}
			}

			return false

		case nil:
			stack = stack[:len(stack)-1]

		default:
			stack = append(stack, element)
		}

		return true
	})

	if !foundReturn || parameterList == nil {
		return nil
	}

	return func() []*protocol.CodeAction {

		var title string
		var textEdit protocol.TextEdit

		if isEmptyType(returnTypeAnnotation.Type) {

			title = fmt.Sprintf("Add return type `%s`", err.ActualType)
			insertionPos := parameterList.EndPosition().Shifted(1)
			textEdit = protocol.TextEdit{
				Range: protocol.Range{
					Start: conversion.ASTToProtocolPosition(insertionPos),
					End:   conversion.ASTToProtocolPosition(insertionPos),
				},
				NewText: fmt.Sprintf(": %s", err.ActualType),
			}
		} else {
			title = fmt.Sprintf("Change return type to `%s`", err.ActualType)
			textEdit = protocol.TextEdit{
				Range: conversion.ASTToProtocolRange(
					returnTypeAnnotation.StartPosition(),
					returnTypeAnnotation.EndPosition(),
				),
				NewText: err.ActualType.String(),
			}
		}

		return []*protocol.CodeAction{
			{
				Title:       title,
				Kind:        protocol.QuickFix,
				Diagnostics: []protocol.Diagnostic{diagnostic},
				Edit: &protocol.WorkspaceEdit{
					Changes: &map[string][]protocol.TextEdit{
						string(uri): {textEdit},
					},
				},
				IsPreferred: true,
			},
		}
	}
}

func isEmptyType(t ast.Type) bool {
	nominalType, ok := t.(*ast.NominalType)
	return ok && nominalType.Identifier.Identifier == ""
}

const indentationCount = 4

func maybeAddMissingMembersCodeActionResolver(
	diagnostic protocol.Diagnostic,
	err *sema.ConformanceError,
	uri protocol.DocumentUri,
) func() []*protocol.CodeAction {

	missingMemberCount := len(err.MissingMembers)
	if missingMemberCount == 0 {
		return nil
	}

	return func() []*protocol.CodeAction {

		var builder strings.Builder

		indentation := strings.Repeat(" ", err.CompositeDeclaration.StartPos.Column+indentationCount)

		for _, missingMember := range err.MissingMembers {
			newMemberSource := formatNewMember(missingMember, indentation)
			if newMemberSource == "" {
				continue
			}

			builder.WriteRune('\n')
			builder.WriteString(indentation)
			if missingMember.Access != ast.AccessNotSpecified {
				builder.WriteString(missingMember.Access.Keyword())
				builder.WriteRune(' ')
			}
			builder.WriteString(newMemberSource)
			builder.WriteRune('\n')
		}

		insertionPos := err.CompositeDeclaration.EndPos

		textEdit := protocol.TextEdit{
			Range: protocol.Range{
				Start: conversion.ASTToProtocolPosition(insertionPos),
				End:   conversion.ASTToProtocolPosition(insertionPos),
			},
			NewText: builder.String(),
		}

		return []*protocol.CodeAction{
			{
				Title:       "Add missing members",
				Kind:        protocol.QuickFix,
				Diagnostics: []protocol.Diagnostic{diagnostic},
				Edit: &protocol.WorkspaceEdit{
					Changes: &map[string][]protocol.TextEdit{
						string(uri): {textEdit},
					},
				},
				IsPreferred: true,
			},
		}
	}
}

func formatNewMember(member *sema.Member, indentation string) string {
	switch member.DeclarationKind {
	case common.DeclarationKindField:
		return fmt.Sprintf(
			"%s %s: %s",
			member.VariableKind.Keyword(),
			member.Identifier.Identifier,
			member.TypeAnnotation,
		)

	case common.DeclarationKindFunction:
		invokableType, ok := member.TypeAnnotation.Type.(sema.InvokableType)
		if !ok {
			return ""
		}

		functionType := invokableType.InvocationFunctionType()

		var parametersBuilder strings.Builder

		for i, parameter := range functionType.Parameters {
			if i > 0 {
				parametersBuilder.WriteString(", ")
			}
			parametersBuilder.WriteString(parameter.QualifiedString())
		}

		var returnType string
		returnTypeAnnotation := functionType.ReturnTypeAnnotation
		if returnTypeAnnotation != nil && returnTypeAnnotation.Type != sema.VoidType {
			returnType = fmt.Sprintf(": %s", returnTypeAnnotation.QualifiedString())
		}

		innerIndentation := strings.Repeat(" ", indentationCount)

		return fmt.Sprintf(
			"fun %s(%s)%s {\n%[4]s%[5]spanic(\"TODO\")\n%[4]s}",
			member.Identifier.Identifier,
			parametersBuilder.String(),
			returnType,
			indentation,
			innerIndentation,
		)

	default:
		return ""
	}
}

func (s *Server) maybeAddVariableDeclarationActionsResolver(
	diagnostic protocol.Diagnostic,
	err *sema.NotDeclaredError,
	uri protocol.DocumentUri,
) func() []*protocol.CodeAction {

	if err.ExpectedKind != common.DeclarationKindVariable {
		return nil
	}

	return func() []*protocol.CodeAction {

		document, ok := s.documents[uri]
		if !ok {
			return nil
		}

		checker := s.checkerForDocument(uri)
		if checker == nil {
			return nil
		}

		var isAssignment bool

		var stack []ast.Element
		ast.Inspect(checker.Program, func(element ast.Element) bool {

			switch element {
			case err.Expression:

				parent := stack[len(stack)-1]
				switch parent := parent.(type) {
				case *ast.AssignmentStatement:
					isAssignment = parent.Target == err.Expression
				}

				return false
			case nil:
				stack = stack[:len(stack)-1]
			default:
				stack = append(stack, element)
			}

			return true
		})

		lineStart := err.Pos.Offset - err.Pos.Column
		indentationEnd := lineStart
		for ; indentationEnd < err.Pos.Offset; indentationEnd++ {
			switch document.Text[indentationEnd] {
			case ' ', '\t':
				continue
			}
			break
		}

		codeActions := make([]*protocol.CodeAction, 0, len(ast.VariableKinds))

		for _, variableKind := range ast.VariableKinds {

			var isPreferred bool
			if isAssignment {
				isPreferred = variableKind == ast.VariableKindVariable
				if variableKind == ast.VariableKindConstant {
					continue
				}
			} else {
				isPreferred = variableKind == ast.VariableKindConstant
			}

			insertionPos := ast.Position{
				Line:   err.Pos.Line,
				Column: 0,
			}

			// TODO: hope for https://github.com/microsoft/language-server-protocol/issues/724
			//  to get implemented, so the cursor can be placed at the value,
			//  or if insertion for a snippet gets supported add a var/let option and a value placeholder

			textEdit := protocol.TextEdit{
				Range: protocol.Range{
					Start: conversion.ASTToProtocolPosition(insertionPos),
					End:   conversion.ASTToProtocolPosition(insertionPos),
				},
				NewText: fmt.Sprintf(
					"%s%s %s = TODO\n",
					document.Text[lineStart:indentationEnd],
					variableKind.Keyword(),
					err.Name,
				),
			}

			codeActions = append(codeActions, &protocol.CodeAction{
				Title:       fmt.Sprintf("Declare %s", variableKind.Name()),
				Kind:        protocol.QuickFix,
				Diagnostics: []protocol.Diagnostic{diagnostic},
				Edit: &protocol.WorkspaceEdit{
					Changes: &map[string][]protocol.TextEdit{
						string(uri): {textEdit},
					},
				},
				IsPreferred: isPreferred,
			})
		}

		return codeActions
	}
}

// convertHint converts a checker hint to a diagnostic
// and an optional code action to resolve the hint.
//
func convertHint(
	hint sema.Hint,
	uri protocol.DocumentUri,
) (
	protocol.Diagnostic,
	func() []*protocol.CodeAction,
) {
	startPosition := hint.StartPosition()
	endPosition := hint.EndPosition()

	protocolRange := conversion.ASTToProtocolRange(startPosition, endPosition)

	diagnostic := protocol.Diagnostic{
		Message: hint.Hint(),
		// protocol.SeverityHint doesn't look prominent enough in VS Code,
		// only the first character of the range is highlighted.
		Severity: protocol.SeverityInformation,
		Range:    protocolRange,
	}

	var codeActionsResolver func() []*protocol.CodeAction

	switch hint := hint.(type) {
	case *sema.ReplacementHint:
		codeActionsResolver = func() []*protocol.CodeAction {
			replacement := hint.Expression.String()
			return []*protocol.CodeAction{
				{
					Title:       fmt.Sprintf("Replace with suggestion `%s`", replacement),
					Kind:        protocol.QuickFix,
					Diagnostics: []protocol.Diagnostic{diagnostic},
					Edit: &protocol.WorkspaceEdit{
						Changes: &map[string][]protocol.TextEdit{
							string(uri): {
								{
									Range:   protocolRange,
									NewText: replacement,
								},
							},
						},
					},
					IsPreferred: true,
				},
			}
		}

	case *sema.RemovalHint:
		codeActionsResolver = func() []*protocol.CodeAction {
			return []*protocol.CodeAction{
				{
					Title:       "Remove unnecessary code",
					Kind:        protocol.QuickFix,
					Diagnostics: []protocol.Diagnostic{diagnostic},
					Edit: &protocol.WorkspaceEdit{
						Changes: &map[string][]protocol.TextEdit{
							string(uri): {
								{
									Range:   protocolRange,
									NewText: "",
								},
							},
						},
					},
					IsPreferred: true,
				},
			}
		}
	}

	return diagnostic, codeActionsResolver
}
