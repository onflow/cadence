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

package server

import (
	"bufio"
	json2 "encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/onflow/cadence/encoding/json"
	"github.com/onflow/cadence/runtime"
	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/parser"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/stdlib"
	"github.com/onflow/cadence/tools/analysis"
	"golang.org/x/exp/maps"

	"github.com/onflow/cadence/languageserver/conversion"
	"github.com/onflow/cadence/languageserver/jsonrpc2"
	"github.com/onflow/cadence/languageserver/protocol"

	linter "github.com/onflow/cadence-lint/analyzers"
)

var functionDeclarations = append(
	stdlib.FlowBuiltInFunctions(stdlib.FlowBuiltinImpls{}),
	stdlib.BuiltinFunctions...,
).ToSemaValueDeclarations()

var valueDeclarations = append(
	functionDeclarations,
	stdlib.BuiltinValues.ToSemaValueDeclarations()...,
)

var typeDeclarations = append(
	stdlib.FlowBuiltInTypes,
	stdlib.BuiltinTypes...,
).ToTypeDeclarations()

// Document represents an open document on the client. It contains all cached
// information about each document that is used to support CodeLens,
// transaction submission, and script execution.
type Document struct {
	Text    string
	Version int32
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
type CommandHandler func(args ...json2.RawMessage) (interface{}, error)

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
type CodeLensProvider func(uri protocol.DocumentURI, version int32, checker *sema.Checker) ([]*protocol.CodeLens, error)

// DiagnosticProvider is a function that is used to provide diagnostics for the given checker
//
type DiagnosticProvider func(uri protocol.DocumentURI, version int32, checker *sema.Checker) ([]protocol.Diagnostic, error)

// DocumentSymbolProvider is a function that is used to provide document symbols for the given checker
//
type DocumentSymbolProvider func(uri protocol.DocumentURI, version int32, checker *sema.Checker) ([]*protocol.DocumentSymbol, error)

// InitializationOptionsHandler is a function that is used to handle initialization options sent by the client
//
type InitializationOptionsHandler func(initializationOptions any) error

type Server struct {
	protocolServer       *protocol.Server
	checkers             map[common.LocationID]*sema.Checker
	documents            map[protocol.DocumentURI]Document
	memberResolvers      map[protocol.DocumentURI]map[string]sema.MemberResolver
	ranges               map[protocol.DocumentURI]map[string]sema.Range
	codeActionsResolvers map[protocol.DocumentURI]map[uuid.UUID]func() []*protocol.CodeAction
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
	// reportCrashes decides when the crash is detected should it be reported
	reportCrashes bool
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
		documents:            make(map[protocol.DocumentURI]Document),
		memberResolvers:      make(map[protocol.DocumentURI]map[string]sema.MemberResolver),
		ranges:               make(map[protocol.DocumentURI]map[string]sema.Range),
		codeActionsResolvers: make(map[protocol.DocumentURI]map[uuid.UUID]func() []*protocol.CodeAction),
		commands:             make(map[string]CommandHandler),
	}
	server.protocolServer = protocol.NewServer(server)

	// init crash reporting
	defer sentry.Flush(2 * time.Second)
	defer sentry.Recover()
	initCrashReporting(server)

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

func (s *Server) checkerForDocument(uri protocol.DocumentURI) *sema.Checker {
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
			CodeLensProvider: protocol.CodeLensOptions{
				ResolveProvider: false,
			},
			CompletionProvider: protocol.CompletionOptions{
				TriggerCharacters: []string{"."},
				ResolveProvider:   true,
			},
			DocumentHighlightProvider: true,
			DocumentSymbolProvider:    true,
			RenameProvider:            true,
			SignatureHelpProvider: protocol.SignatureHelpOptions{
				TriggerCharacters: []string{"("},
			},
			CodeActionProvider: true,
			InlayHintProvider:  true,
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

// initCrashReporting set-ups sentry as crash reporting tool, it also sets listener for panics.
func initCrashReporting(server *Server) {
	sentrySyncTransport := sentry.NewHTTPSyncTransport()
	sentrySyncTransport.Timeout = time.Second * 3

	_ = sentry.Init(sentry.ClientOptions{
		Dsn:              "https://7725b80e4d5a4625845270176a6d8bd5@o114654.ingest.sentry.io/6330569",
		AttachStacktrace: true,
		Transport:        sentrySyncTransport,
		BeforeSend: func(event *sentry.Event, hint *sentry.EventHint) *sentry.Event {
			if server.reportCrashes {
				return event
			}

			return nil
		},
	})
}

const (
	accessCheckModeOption = "accessCheckMode"
	reportCrashesOption   = "reportCrashes"
)

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

func (s *Server) configure(opts any) {
	optsMap, ok := opts.(map[string]any)
	if !ok {
		return
	}

	if accessCheckModeName, ok := optsMap[accessCheckModeOption].(string); ok {
		s.accessCheckMode = accessCheckModeFromName(accessCheckModeName)
	} else {
		s.accessCheckMode = sema.AccessCheckModeStrict
	}

	if reportCrashesValue, ok := optsMap[reportCrashesOption].(bool); ok {
		s.reportCrashes = reportCrashesValue
	} else {
		s.reportCrashes = true // report by default
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
				RegisterOptions: protocol.ExecuteCommandOptions{
					Commands: commands,
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
	URI protocol.DocumentURI `json:"uri"`

	Valid bool `json:"valid"`
}

const cadenceCheckCompletedMethodName = "cadence/checkCompleted"

func (s *Server) checkAndPublishDiagnostics(
	conn protocol.Conn,
	uri protocol.DocumentURI,
	text string,
	version int32,
) {

	diagnostics, _ := s.getDiagnostics(uri, text, version, conn.LogMessage)

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

	var markup strings.Builder

	_, _ = fmt.Fprintf(
		&markup,
		"**Type**\n\n```cadence\n%s\n```\n",
		documentType(occurrence.Origin.Type),
	)

	docString := occurrence.Origin.DocString
	if docString != "" {
		_, _ = fmt.Fprintf(
			&markup,
			"\n**Documentation**\n\n%s\n",
			docString,
		)
	}

	contents := protocol.MarkupContent{
		Kind:  protocol.Markdown,
		Value: markup.String(),
	}
	return &protocol.Hover{Contents: contents}, nil
}

func documentType(ty sema.Type) string {
	if functionType, ok := ty.(*sema.FunctionType); ok {
		return documentFunctionType(functionType)
	}
	return ty.QualifiedString()
}

func documentFunctionType(ty *sema.FunctionType) string {
	var builder strings.Builder
	builder.WriteString("fun ")
	if len(ty.TypeParameters) > 0 {
		builder.WriteRune('<')
		for i, typeParameter := range ty.TypeParameters {
			if i > 0 {
				builder.WriteString(", ")
			}
			builder.WriteString(typeParameter.QualifiedString())
		}
		builder.WriteRune('>')
	}
	builder.WriteRune('(')
	for i, parameter := range ty.Parameters {
		if i > 0 {
			builder.WriteString(", ")
		}
		builder.WriteString(parameter.QualifiedString())
	}
	builder.WriteString(")")

	if ty.ReturnTypeAnnotation.Type != sema.VoidType {
		builder.WriteString(": ")
		builder.WriteString(ty.ReturnTypeAnnotation.QualifiedString())
	}
	return builder.String()
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

	var activeParameter uint32

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
		ActiveParameter: activeParameter,
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
		Changes: map[protocol.DocumentURI][]protocol.TextEdit{
			uri: textEdits,
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
	URI protocol.DocumentURI `json:"uri"`
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
	uri protocol.DocumentURI,
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
	uri protocol.DocumentURI,
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

			if functionType, ok := r.Type.(*sema.FunctionType); ok && functionType.IsConstructor {
				item.Kind = protocol.ConstructorCompletion

				s.prepareParametersCompletionItem(
					item,
					r.Identifier,
					functionType.Parameters,
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
	member := resolver.Resolve(nil, item.Label, ast.Range{}, func(err error) { /* NO-OP */ })
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

func (s *Server) maybeResolveMember(uri protocol.DocumentURI, id string, result *protocol.CompletionItem) bool {
	memberResolvers, ok := s.memberResolvers[uri]
	if !ok {
		return false
	}

	resolver, ok := memberResolvers[id]
	if !ok {
		return false
	}

	member := resolver.Resolve(nil, result.Label, ast.Range{}, func(err error) { /* NO-OP */ })

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

func (s *Server) maybeResolveRange(uri protocol.DocumentURI, id string, result *protocol.CompletionItem) bool {
	ranges, ok := s.ranges[uri]
	if !ok {
		return false
	}

	r, ok := ranges[id]
	if !ok {
		return false
	}

	if functionType, ok := r.Type.(*sema.FunctionType); ok && functionType.IsConstructor {
		typeString := functionType.QualifiedString()

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
func (s *Server) ExecuteCommand(conn protocol.Conn, params *protocol.ExecuteCommandParams) (any, error) {
	conn.LogMessage(&protocol.LogMessageParams{
		Type:    protocol.Log,
		Message: fmt.Sprintf("called execute command: %s", params.Command),
	})

	commandHandler, ok := s.commands[params.Command]
	if !ok {
		return nil, fmt.Errorf("invalid command: %s", params.Command)
	}

	res, err := commandHandler(params.Arguments...)
	if err != nil {
		return nil, err
	}

	return res, nil
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

func (s *Server) DocumentLink(
	_ protocol.Conn,
	_ *protocol.DocumentLinkParams,
) (
	symbols []*protocol.DocumentLink,
	err error,
) {
	return
}

func (s *Server) InlayHint(
	_ protocol.Conn,
	params *protocol.InlayHintParams,
) (
	inlayHints []*protocol.InlayHint,
	err error,
) {

	// NOTE: Always initialize to an empty slice, i.e DON'T use nil:
	// The later will be ignored instead of being treated as no items
	inlayHints = []*protocol.InlayHint{}

	// get uri from parameters caught by grpc server
	uri := params.TextDocument.URI
	checker := s.checkerForDocument(uri)
	if checker == nil {
		return
	}

	var variableDeclarations []*ast.VariableDeclaration

	ast.Inspect(checker.Program, func(element ast.Element) bool {

		variableDeclaration, ok := element.(*ast.VariableDeclaration)
		if !ok || variableDeclaration.TypeAnnotation != nil {
			return true
		}

		variableDeclarations = append(variableDeclarations, variableDeclaration)

		return true
	})

	for _, variableDeclaration := range variableDeclarations {
		targetType := checker.Elaboration.VariableDeclarationTargetTypes[variableDeclaration]
		identifierEndPosition := variableDeclaration.Identifier.EndPosition(nil)
		inlayHintPosition := conversion.ASTToProtocolPosition(identifierEndPosition.Shifted(nil, 1))
		inlayHint := protocol.InlayHint{
			Position: &inlayHintPosition,
			Label: []protocol.InlayHintLabelPart{
				{
					Value: fmt.Sprintf(": %s", targetType.QualifiedString()),
				},
			},
			Kind: protocol.Type,
		}

		inlayHints = append(inlayHints, &inlayHint)
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

var lintingAnalyzers = maps.Values(linter.Analyzers)

// getDiagnostics parses and checks the given file and generates diagnostics
// indicating each syntax or semantic error. Returns a list of diagnostics
// that the caller is responsible for publishing to the client.
//
// Returns an error if an unexpected error occurred.
func (s *Server) getDiagnostics(
	uri protocol.DocumentURI,
	text string,
	version int32,
	log func(*protocol.LogMessageParams),
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

	program, parseError := parse(text, string(uri), log)

	// If there were parsing errors, convert each one to a diagnostic and exit
	// without checking.

	if parseError != nil {
		if parentErr, ok := parseError.(errors.ParentError); ok {
			parserDiagnostics := s.getDiagnosticsForParentError(uri, parentErr, codeActionsResolvers, log)
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
		nil,
		true,
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
						if importedProgram == nil {
							return nil, &sema.CheckerError{
								Errors: []error{fmt.Errorf("cannot import %s", importedLocation)},
							}
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
	log(&protocol.LogMessageParams{
		Type:    protocol.Info,
		Message: fmt.Sprintf("checking %s took %s", string(uri), elapsed),
	})

	s.checkers[location.ID()] = checker

	if checkError != nil {
		if parentErr, ok := checkError.(errors.ParentError); ok {
			checkerDiagnostics := s.getDiagnosticsForParentError(uri, parentErr, codeActionsResolvers, log)
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

	if checkError != nil {
		return
	}

	analysisProgram := analysis.Program{
		Program:     program,
		Elaboration: checker.Elaboration,
		Location:    checker.Location,
		Code:        text,
	}

	var reportLock sync.Mutex

	report := func(linterDiagnostic analysis.Diagnostic) {
		reportLock.Lock()
		defer reportLock.Unlock()
		diagnostic, codeActionsResolver := convertDiagnostic(linterDiagnostic, uri)
		if codeActionsResolver != nil {
			codeActionsResolverID := uuid.New()
			diagnostic.Data = codeActionsResolverID
			codeActionsResolvers[codeActionsResolverID] = codeActionsResolver
		}
		diagnostics = append(diagnostics, diagnostic)
	}

	analysisProgram.Run(lintingAnalyzers, report)

	return
}

// getDiagnosticsForParentError unpacks all child errors and converts each to
// a diagnostic. Both parser and checker errors can be unpacked.
//
// Logs any conversion failures to the client.
func (s *Server) getDiagnosticsForParentError(
	uri protocol.DocumentURI,
	err errors.ParentError,
	codeActionsResolvers map[uuid.UUID]func() []*protocol.CodeAction,
	log func(*protocol.LogMessageParams),
) (
	diagnostics []protocol.Diagnostic,
) {
	for _, childErr := range err.ChildErrors() {
		convertibleErr, ok := childErr.(convertibleError)
		if !ok {
			log(&protocol.LogMessageParams{
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
					endPos := positioned.EndPosition(nil)
					message := errorNote.Message()

					diagnostic.RelatedInformation = append(
						diagnostic.RelatedInformation,
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
func parse(code, location string, log func(*protocol.LogMessageParams)) (*ast.Program, error) {
	start := time.Now()
	program, err := parser.ParseProgram(code, nil)
	elapsed := time.Since(start)

	log(&protocol.LogMessageParams{
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

	return parser.ParseProgram(code, nil)
}

func (s *Server) GetDocument(uri protocol.DocumentURI) (doc Document, ok bool) {
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
func (s *Server) getEntryPointParameters(args ...json2.RawMessage) (interface{}, error) {

	err := CheckCommandArgumentCount(args, 1)
	if err != nil {
		return nil, err
	}

	var uriArg string
	err = json2.Unmarshal(args[0], &uriArg)
	if err != nil {
		return nil, fmt.Errorf("invalid URI argument: %#+v: %w", args[0], err)
	}

	uri := protocol.DocumentURI(uriArg)
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
func (s *Server) getContractInitializerParameters(args ...json2.RawMessage) (interface{}, error) {

	err := CheckCommandArgumentCount(args, 1)
	if err != nil {
		return nil, err
	}

	var uriArg string
	err = json2.Unmarshal(args[0], &uriArg)
	if err != nil {
		return nil, fmt.Errorf("invalid URI argument: %#+v: %w", args[0], err)
	}

	uri := protocol.DocumentURI(uriArg)
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
func (s *Server) parseEntryPointArguments(args ...json2.RawMessage) (interface{}, error) {

	err := CheckCommandArgumentCount(args, 2)
	if err != nil {
		return nil, err
	}

	var uriArg string
	err = json2.Unmarshal(args[0], &uriArg)
	if err != nil {
		return nil, fmt.Errorf("invalid URI argument: %#+v: %w", args[0], err)
	}

	uri := protocol.DocumentURI(uriArg)
	checker := s.checkerForDocument(uri)
	if checker == nil {
		return nil, fmt.Errorf("could not find document for URI %s", uri)
	}

	var arguments []interface{}
	err = json2.Unmarshal(args[1], &arguments)
	if err != nil {
		return nil, fmt.Errorf("invalid arguments argument: %#+v: %w", args[1], err)
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

	result := make([]any, len(arguments))

	for i, argument := range arguments {
		parameter := parameters[i]
		parameterType := parameter.TypeAnnotation.Type

		argumentCode, ok := argument.(string)
		if !ok {
			return nil, fmt.Errorf("invalid argument at index %d: %#+v", i, argument)
		}
		value, err := runtime.ParseLiteral(argumentCode, parameterType, nil)
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

type insertionPosition struct {
	ast.Position
	before bool
}

// convertError converts a checker error to a diagnostic
// and an optional code action to resolve the error.
//
func (s *Server) convertError(
	err convertibleError,
	uri protocol.DocumentURI,
) (
	protocol.Diagnostic,
	func() []*protocol.CodeAction,
) {
	startPosition := err.StartPosition()
	endPosition := err.EndPosition(nil)

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
		codeActionsResolver = s.maybeReturnTypeChangeCodeActionsResolver(diagnostic, uri, err)

	case *sema.ConformanceError:
		codeActionsResolver = maybeAddMissingMembersCodeActionResolver(diagnostic, err, uri)

	case *sema.NotDeclaredError:
		if err.ExpectedKind == common.DeclarationKindVariable {
			codeActionsResolver = s.maybeAddDeclarationActionsResolver(
				diagnostic,
				uri,
				err.Expression,
				err.Pos,
				err.Name,
				nil,
			)
		}

	case *sema.NotDeclaredMemberError:
		var declarationGetter func(elaboration *sema.Elaboration) ast.Declaration

		switch ty := err.Type.(type) {
		case *sema.CompositeType:
			declarationGetter = func(elaboration *sema.Elaboration) ast.Declaration {
				return elaboration.CompositeTypeDeclarations[ty]
			}
		case *sema.InterfaceType:
			declarationGetter = func(elaboration *sema.Elaboration) ast.Declaration {
				return elaboration.InterfaceTypeDeclarations[ty]
			}
		}

		if declarationGetter != nil {
			codeActionsResolver = s.maybeAddDeclarationActionsResolver(
				diagnostic,
				uri,
				err.Expression,
				err.StartPos,
				err.Name,
				func(checker *sema.Checker, isFunction bool) insertionPosition {
					declaration := declarationGetter(checker.Elaboration)

					members := declaration.DeclarationMembers()
					declarations := members.Declarations()
					functions := members.Functions()
					fields := members.Fields()

					switch {
					case isFunction && len(functions) > 0:
						// If a function is inserted,
						// prefer adding it after the last function (if any)
						lastFunction := functions[len(functions)-1]
						return insertionPosition{
							before:   false,
							Position: lastFunction.EndPosition(nil).Shifted(nil, 1),
						}
					case !isFunction && len(fields) > 0:
						// If a field is inserted,
						// prefer inserting it after the last field (if any)
						lastField := fields[len(fields)-1]
						return insertionPosition{
							before:   false,
							Position: lastField.EndPosition(nil).Shifted(nil, 1),
						}
					case !isFunction && len(functions) > 0:
						// If a field is inserted,
						// and there are no fields, but functions,
						// insert it before the first function
						firstFunction := functions[0]
						return insertionPosition{
							before:   true,
							Position: firstFunction.StartPosition(),
						}
					}

					// By default, insert the declaration after the last declaration (if any).
					// Otherwise, insert it before the end of the containing declaration

					if len(declarations) > 0 {
						lastDeclaration := declarations[len(declarations)-1]
						return insertionPosition{
							before:   false,
							Position: lastDeclaration.EndPosition(nil).Shifted(nil, 1),
						}
					}

					return insertionPosition{
						before:   true,
						Position: declaration.EndPosition(nil),
					}
				},
			)
		}
	}

	return diagnostic, codeActionsResolver
}

func (s *Server) maybeReturnTypeChangeCodeActionsResolver(
	diagnostic protocol.Diagnostic,
	uri protocol.DocumentURI,
	err *sema.TypeMismatchError,
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

			// We found the error expression.
			// Determine what should be declared based on the context,
			// i.e. from the parents

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
			insertionPos := parameterList.EndPosition(nil).Shifted(nil, 1)
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
					returnTypeAnnotation.EndPosition(nil),
				),
				NewText: err.ActualType.String(),
			}
		}

		return []*protocol.CodeAction{
			{
				Title:       title,
				Kind:        protocol.QuickFix,
				Diagnostics: []protocol.Diagnostic{diagnostic},
				Edit: protocol.WorkspaceEdit{
					Changes: map[protocol.DocumentURI][]protocol.TextEdit{
						uri: {textEdit},
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
	uri protocol.DocumentURI,
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
				Edit: protocol.WorkspaceEdit{
					Changes: map[protocol.DocumentURI][]protocol.TextEdit{
						uri: {textEdit},
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
		functionType, ok := member.TypeAnnotation.Type.(*sema.FunctionType)
		if !ok {
			return ""
		}

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

func (s *Server) maybeAddDeclarationActionsResolver(
	diagnostic protocol.Diagnostic,
	uri protocol.DocumentURI,
	errorExpression ast.Expression,
	errorPos ast.Position,
	name string,
	memberInsertionPosGetter func(checker *sema.Checker, isFunction bool) insertionPosition,
) func() []*protocol.CodeAction {

	return func() []*protocol.CodeAction {

		document, ok := s.documents[uri]
		if !ok {
			return nil
		}

		checker := s.checkerForDocument(uri)
		if checker == nil {
			return nil
		}

		var isAssignmentTarget bool
		var isInvoked bool
		var parentFunctionEndPos *ast.Position
		var invocationArgumentTypes []sema.Type
		var invocationArgumentLabels []string
		var invocationReturnType sema.Type

		var stack []ast.Element
		ast.Inspect(checker.Program, func(element ast.Element) bool {

			switch element {
			case errorExpression:

				// We found the error expression.
				// Determine what should be declared based on the context,
				// i.e. from the parents

				parent := stack[len(stack)-1]
				switch parent := parent.(type) {
				case *ast.AssignmentStatement:
					isAssignmentTarget = parent.Target == errorExpression

				case *ast.InvocationExpression:
					isInvoked = parent.InvokedExpression == errorExpression

					invocationArgumentTypes = checker.Elaboration.InvocationExpressionArgumentTypes[parent]
					invocationReturnType = checker.Elaboration.InvocationExpressionReturnTypes[parent]

					invocationArgumentLabels = make([]string, 0, len(parent.Arguments))
					for _, argument := range parent.Arguments {
						invocationArgumentLabels = append(
							invocationArgumentLabels,
							argument.Label,
						)
					}

					if memberInsertionPosGetter == nil {

						// Find the containing function declaration, if any
						for i := len(stack) - 2; i > 0; i-- {
							element := stack[i]
							switch element := element.(type) {
							case *ast.FunctionDeclaration:
								position := element.EndPosition(nil)
								parentFunctionEndPos = &position
								break

							case *ast.SpecialFunctionDeclaration:
								position := element.FunctionDeclaration.EndPosition(nil)
								parentFunctionEndPos = &position
								break
							}
						}
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

		var insertionPos insertionPosition

		if isInvoked {

			// If the identifier is invoked,
			// propose the declaration of a function

			if memberInsertionPosGetter != nil {
				insertionPos = memberInsertionPosGetter(checker, true)
			} else {

				// If the function declaration is not a member,
				// declare it after the parent function (if any).
				//
				// If there is no parent function,
				// then the declaration is local,
				// so insert the function before the use

				if parentFunctionEndPos != nil {
					insertionPos = insertionPosition{
						before:   false,
						Position: parentFunctionEndPos.Shifted(nil, 1),
					}
				} else {
					insertionPos = insertionPosition{
						before: true,
						Position: ast.Position{
							Line:   errorPos.Line,
							Column: 0,
						},
					}
				}
			}

			return functionDeclarationCodeActions(
				uri,
				document,
				diagnostic,
				insertionPos,
				name,
				invocationArgumentTypes,
				invocationArgumentLabels,
				invocationReturnType,
			)

		} else {

			// If the identifier is not invoked,
			// propose the declaration of a variable,
			// or a constant (if the identifier is not assigned to)

			if memberInsertionPosGetter != nil {
				insertionPos = memberInsertionPosGetter(checker, false)

				memberExpression := errorExpression.(*ast.MemberExpression)
				expectedType := checker.Elaboration.MemberExpressionExpectedTypes[memberExpression]

				var typeString string
				if expectedType != nil {
					typeString = expectedType.QualifiedString()
				} else {
					typeString = "TODO"
				}

				return fieldDeclarationCodeActions(
					uri,
					document,
					diagnostic,
					insertionPos,
					name,
					typeString,
					isAssignmentTarget,
				)
			} else {

				// If a variable is inserted,
				// then insert it before the error's line

				insertionPos := ast.Position{
					Offset: errorPos.Offset - errorPos.Column,
					Line:   errorPos.Line,
					Column: 0,
				}

				for insertionPos.Column < errorPos.Column {
					switch document.Text[insertionPos.Offset] {
					case ' ', '\t':
						insertionPos.Offset++
						insertionPos.Column++
						continue
					}
					break
				}

				return variableDeclarationCodeActions(
					uri,
					document,
					diagnostic,
					insertionPos,
					name,
					isAssignmentTarget,
				)
			}
		}
	}
}

func extractIndentation(text string, pos ast.Position) string {
	lineStartOffset := pos.Offset - pos.Column
	indentationEndOffset := lineStartOffset
	for ; indentationEndOffset < pos.Offset; indentationEndOffset++ {
		switch text[indentationEndOffset] {
		case ' ', '\t':
			continue
		}
		break
	}
	return text[lineStartOffset:indentationEndOffset]
}

func functionDeclarationCodeActions(
	uri protocol.DocumentURI,
	document Document,
	diagnostic protocol.Diagnostic,
	insertionPos insertionPosition,
	name string,
	invocationArgumentTypes []sema.Type,
	invocationArgumentLabels []string,
	invocationReturnType sema.Type,
) []*protocol.CodeAction {

	pos := insertionPos.Position

	indentation := extractIndentation(document.Text, pos)

	insertionRange := protocol.Range{
		Start: conversion.ASTToProtocolPosition(pos),
		End:   conversion.ASTToProtocolPosition(pos),
	}

	var parameters strings.Builder

	for i, argumentType := range invocationArgumentTypes {

		// Only support the generation of parameters from a-z
		if i > 'z' {
			break
		}

		if i > 0 {
			parameters.WriteString(", ")
		}
		argumentLabel := invocationArgumentLabels[i]
		if argumentLabel == "" {
			parameters.WriteString("_ ")
			// Generate a parameter name (a-z)
			parameters.WriteByte(byte('a' + i))
		} else {
			parameters.WriteString(argumentLabel)
		}
		parameters.WriteString(": ")
		if argumentType.IsInvalidType() {
			parameters.WriteString(sema.VoidType.String())
		} else {
			parameters.WriteString(argumentType.QualifiedString())
		}
	}

	var returnType string
	if invocationReturnType != nil &&
		!invocationReturnType.IsInvalidType() &&
		invocationReturnType != sema.VoidType {

		returnType = fmt.Sprintf(": %s", invocationReturnType.QualifiedString())
	}

	prefix, suffix := insertionPrefixSuffix(insertionPos, document, indentation)

	textEdit := protocol.TextEdit{
		Range: insertionRange,
		NewText: fmt.Sprintf(
			"%sfun %s(%s)%s {}\n%s",
			prefix,
			name,
			parameters.String(),
			returnType,
			suffix,
		),
	}

	return []*protocol.CodeAction{
		{
			Title:       "Declare function",
			Kind:        protocol.QuickFix,
			Diagnostics: []protocol.Diagnostic{diagnostic},
			Edit: protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentURI][]protocol.TextEdit{
					uri: {textEdit},
				},
			},
			IsPreferred: true,
		},
	}
}

func insertionPrefixSuffix(
	insertionPos insertionPosition,
	document Document,
	indentation string,
) (
	prefix string,
	suffix string,
) {
	if insertionPos.before {

		for offset := insertionPos.Offset - 1; offset >= insertionPos.Offset-insertionPos.Column; offset-- {
			switch document.Text[offset] {
			case ' ', '\t':
				continue
			case '{':
				prefix = "\n" + indentation
				break
			default:
				break
			}
		}

		if document.Text[insertionPos.Offset] == '}' {
			prefix += "    "
		}
		suffix = "\n" + indentation
	} else {
		prefix = "\n\n" + indentation
	}
	return prefix, suffix
}

func variableDeclarationCodeActions(
	uri protocol.DocumentURI,
	document Document,
	diagnostic protocol.Diagnostic,
	insertionPos ast.Position,
	name string,
	isAssignmentTarget bool,
) []*protocol.CodeAction {

	codeActions := make([]*protocol.CodeAction, 0, len(ast.VariableKinds))

	indentation := extractIndentation(document.Text, insertionPos)

	insertionRange := protocol.Range{
		Start: conversion.ASTToProtocolPosition(insertionPos),
		End:   conversion.ASTToProtocolPosition(insertionPos),
	}

	for _, variableKind := range ast.VariableKinds {

		var isPreferred bool
		if isAssignmentTarget {
			isPreferred = variableKind == ast.VariableKindVariable
			if variableKind == ast.VariableKindConstant {
				continue
			}
		} else {
			isPreferred = variableKind == ast.VariableKindConstant
		}

		// TODO: hope for https://github.com/microsoft/language-server-protocol/issues/724
		//  to get implemented, so the cursor can be placed at the value,
		//  or if insertion for a snippet gets supported add a var/let option and a value placeholder

		textEdit := protocol.TextEdit{
			Range: insertionRange,
			NewText: fmt.Sprintf(
				"%s %s = TODO\n%s",
				variableKind.Keyword(),
				name,
				indentation,
			),
		}

		codeActions = append(codeActions, &protocol.CodeAction{
			Title:       fmt.Sprintf("Declare %s", variableKind.Name()),
			Kind:        protocol.QuickFix,
			Diagnostics: []protocol.Diagnostic{diagnostic},
			Edit: protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentURI][]protocol.TextEdit{
					uri: {textEdit},
				},
			},
			IsPreferred: isPreferred,
		})
	}

	return codeActions
}

func fieldDeclarationCodeActions(
	uri protocol.DocumentURI,
	document Document,
	diagnostic protocol.Diagnostic,
	insertionPos insertionPosition,
	name string,
	typeString string,
	isAssignmentTarget bool,
) []*protocol.CodeAction {

	pos := insertionPos.Position

	indentation := extractIndentation(document.Text, pos)

	codeActions := make([]*protocol.CodeAction, 0, len(ast.VariableKinds))

	insertionRange := protocol.Range{
		Start: conversion.ASTToProtocolPosition(pos),
		End:   conversion.ASTToProtocolPosition(pos),
	}

	prefix, suffix := insertionPrefixSuffix(insertionPos, document, indentation)

	for _, variableKind := range ast.VariableKinds {

		var isPreferred bool
		if isAssignmentTarget {
			isPreferred = variableKind == ast.VariableKindVariable
			if variableKind == ast.VariableKindConstant {
				continue
			}
		} else {
			isPreferred = variableKind == ast.VariableKindConstant
		}

		// TODO: hope for https://github.com/microsoft/language-server-protocol/issues/724
		//  to get implemented, so the cursor can be placed at the value,
		//  or if insertion for a snippet gets supported add a var/let option and a value placeholder

		textEdit := protocol.TextEdit{
			Range: insertionRange,
			NewText: fmt.Sprintf(
				"%s%s %s: %s\n%s",
				prefix,
				variableKind.Keyword(),
				name,
				typeString,
				suffix,
			),
		}

		codeActions = append(codeActions, &protocol.CodeAction{
			Title:       fmt.Sprintf("Declare %s field", variableKind.Name()),
			Kind:        protocol.QuickFix,
			Diagnostics: []protocol.Diagnostic{diagnostic},
			Edit: protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentURI][]protocol.TextEdit{
					uri: {textEdit},
				},
			},
			IsPreferred: isPreferred,
		})
	}

	return codeActions
}

// convertDiagnostic converts a linter diagnostic to a languagserver
// and an optional code action to resolve the diagnostic.
//
func convertDiagnostic(
	linterDiagnostic analysis.Diagnostic,
	uri protocol.DocumentURI,
) (
	protocol.Diagnostic,
	func() []*protocol.CodeAction,
) {

	protocolRange := conversion.ASTToProtocolRange(linterDiagnostic.StartPos, linterDiagnostic.EndPos)

	var protocolDiagnostic protocol.Diagnostic
	var message string

	var codeActionsResolver func() []*protocol.CodeAction

	switch linterDiagnostic.Category {
	case linter.ReplacementCategory:
		codeActionsResolver = func() []*protocol.CodeAction {
			return []*protocol.CodeAction{
				{
					Title:       fmt.Sprintf("%s `%s`", linterDiagnostic.Message, linterDiagnostic.SecondaryMessage),
					Kind:        protocol.QuickFix,
					Diagnostics: []protocol.Diagnostic{protocolDiagnostic},
					Edit: protocol.WorkspaceEdit{
						Changes: map[protocol.DocumentURI][]protocol.TextEdit{
							uri: {
								{
									Range:   protocolRange,
									NewText: linterDiagnostic.SecondaryMessage,
								},
							},
						},
					},
					IsPreferred: true,
				},
			}
		}
		message = fmt.Sprintf("%s `%s`", linterDiagnostic.Message, linterDiagnostic.SecondaryMessage)
	case linter.RemovalCategory:
		codeActionsResolver = func() []*protocol.CodeAction {
			return []*protocol.CodeAction{
				{
					Title:       "Remove unnecessary code",
					Kind:        protocol.QuickFix,
					Diagnostics: []protocol.Diagnostic{protocolDiagnostic},
					Edit: protocol.WorkspaceEdit{
						Changes: map[protocol.DocumentURI][]protocol.TextEdit{
							uri: {
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

	if message == "" {
		message = linterDiagnostic.Message
	}

	protocolDiagnostic = protocol.Diagnostic{
		Message: message,
		// protocol.SeverityHint doesn't look prominent enough in VS Code,
		// only the first character of the range is highlighted.
		Severity: protocol.SeverityInformation,
		Range:    protocolRange,
	}

	return protocolDiagnostic, codeActionsResolver
}
