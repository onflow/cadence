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
	"path"
	"strconv"
	"strings"
	"time"

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

// InitializationOptionsHandler is a function that is used to handle initialization options sent by the client
//
type InitializationOptionsHandler func(initializationOptions interface{}) error

type Server struct {
	protocolServer  *protocol.Server
	checkers        map[protocol.DocumentUri]*sema.Checker
	documents       map[protocol.DocumentUri]Document
	memberResolvers map[protocol.DocumentUri]map[string]sema.MemberResolver
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
	// initializationOptionsHandlers are the functions that are used to handle initialization options sent by the client
	initializationOptionsHandlers []InitializationOptionsHandler
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
		checkers:        make(map[protocol.DocumentUri]*sema.Checker),
		documents:       make(map[protocol.DocumentUri]Document),
		memberResolvers: make(map[protocol.DocumentUri]map[string]sema.MemberResolver),
		commands:        make(map[string]CommandHandler),
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
		},
	}

	for _, handler := range s.initializationOptionsHandlers {
		err := handler(params.InitializationOptions)
		if err != nil {
			return nil, err
		}
	}

	// after initialization, indicate to the client which commands we support
	go s.registerCommands(conn)

	return result, nil
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

	valid := len(diagnostics) == 0
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
	checker, ok := s.checkers[uri]
	if !ok {
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
) (*protocol.Location, error) {

	uri := params.TextDocument.URI
	checker, ok := s.checkers[uri]
	if !ok {
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

// TODO
func (s *Server) SignatureHelp(
	_ protocol.Conn,
	_ *protocol.TextDocumentPositionParams,
) (*protocol.SignatureHelp, error) {
	return nil, nil
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
	checker, ok := s.checkers[uri]
	if !ok {
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
	_ protocol.Conn,
	params *protocol.CompletionParams,
) (
	items []*protocol.CompletionItem,
	err error,
) {
	// NOTE: Always initialize to an empty slice, i.e DON'T use nil:
	// The later will be ignored instead of being treated as no items
	items = []*protocol.CompletionItem{}

	uri := params.TextDocument.URI
	checker, ok := s.checkers[uri]
	if !ok {
		return
	}

	document, ok := s.documents[uri]
	if !ok {
		return
	}

	position := conversion.ProtocolToSemaPosition(params.Position)

	memberCompletions := s.memberCompletions(position, checker, uri)

	// prioritize member completion items over other items
	for _, item := range memberCompletions {
		item.SortText = fmt.Sprintf("1" + item.Label)
	}

	items = append(items, memberCompletions...)

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
			},
		}

		// If the member is a function, also prepare the argument list
		// with placeholders and suggest it

		if resolver.Kind == common.DeclarationKindFunction {
			s.prepareFunctionMemberCompletionItem(item, resolver, name)
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

	item.InsertTextFormat = protocol.SnippetTextFormat

	var builder strings.Builder
	builder.WriteString(name)
	builder.WriteRune('(')

	for i, parameter := range functionType.Parameters {
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
		common.DeclarationKindContract:
		return protocol.ClassCompletion

	case common.DeclarationKindStructureInterface,
		common.DeclarationKindResourceInterface,
		common.DeclarationKindContractInterface:
		return protocol.InterfaceCompletion

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

// Completion is called to compute completion items at a given cursor position.
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

	memberResolvers, ok := s.memberResolvers[data.URI]
	if !ok {
		return
	}

	resolver, ok := memberResolvers[item.Label]
	if !ok {
		return
	}

	member := resolver.Resolve(item.Label, ast.Range{}, func(err error) { /* NO-OP */ })

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

	return
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
const inMemoryPrefix = "inmemory:/"

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

	// NOTE: Always initialize to an empty slice, i.e DON'T use nil:
	// The later will be ignored instead of being treated as no items
	diagnostics = []protocol.Diagnostic{}

	program, parseError := parse(conn, text, string(uri))

	// If there were parsing errors, convert each one to a diagnostic and exit
	// without checking.

	if parseError != nil {
		if parentErr, ok := parseError.(errors.ParentError); ok {
			parserDiagnostics := getDiagnosticsForParentError(conn, uri, parentErr)
			diagnostics = append(diagnostics, parserDiagnostics...)
		}
	}

	// If there is a parse result succeeded proceed with resolving imports and checking the parsed program,
	// even if there there might have been parsing errors.

	if program == nil {
		delete(s.checkers, uri)
		return
	}

	mainPath := string(uri)

	if strings.HasPrefix(mainPath, filePrefix) {
		mainPath = mainPath[len(filePrefix):]
	} else if strings.HasPrefix(mainPath, inMemoryPrefix) {
		mainPath = mainPath[len(inMemoryPrefix):]
	}

	var checker *sema.Checker
	checker, diagnosticsErr = sema.NewChecker(
		program,
		uriToLocation(uri),
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
		sema.WithOriginsAndOccurrencesEnabled(true),
		sema.WithImportHandler(
			func(checker *sema.Checker, importedLocation common.Location) (sema.Import, error) {

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

					importedProgram, err := s.resolveImport(importedLocation)
					if err != nil {
						return nil, &sema.CheckerError{
							Errors: []error{err},
						}
					}

					importedChecker, err := checker.SubChecker(importedProgram, importedLocation)
					if err != nil {
						return nil, &sema.CheckerError{
							Errors: []error{err},
						}
					}

					err = importedChecker.Check()
					if err != nil {
						return nil, err
					}

					return sema.ElaborationImport{
						Elaboration: importedChecker.Elaboration,
					}, nil
				}
			},
		),
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

	s.checkers[uri] = checker

	if checkError != nil {
		if parentErr, ok := checkError.(errors.ParentError); ok {
			checkerDiagnostics := getDiagnosticsForParentError(conn, uri, parentErr)
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
		diagnostic := convertHint(hint)
		diagnostics = append(diagnostics, diagnostic)
	}

	return
}

// getDiagnosticsForParentError unpacks all child errors and converts each to
// a diagnostic. Both parser and checker errors can be unpacked.
//
// Logs any conversion failures to the client.
func getDiagnosticsForParentError(
	conn protocol.Conn,
	uri protocol.DocumentUri,
	err errors.ParentError,
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
		diagnostic := convertError(convertibleErr)

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

func isPathLocation(location common.Location) bool {
	return locationToPath(location) != ""
}

func normalizePathLocation(base, relative common.Location) common.Location {
	basePath := locationToPath(base)
	relativePath := locationToPath(relative)

	if basePath == "" || relativePath == "" {
		return relative
	}

	normalizedPath := normalizePath(basePath, relativePath)

	return common.StringLocation(normalizedPath)
}

func normalizePath(basePath, relativePath string) string {
	if path.IsAbs(relativePath) {
		return relativePath
	}

	return path.Join(path.Dir(basePath), relativePath)
}

func locationToPath(location common.Location) string {
	stringLocation, ok := location.(common.StringLocation)
	if !ok {
		return ""
	}

	s := string(stringLocation)

	return s
}

func uriToLocation(uri protocol.DocumentUri) common.StringLocation {
	s := string(uri)

	if strings.HasPrefix(s, filePrefix) {
		return common.StringLocation(s[len(filePrefix):])
	}

	return common.StringLocation(s)
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

	uri, ok := args[0].(string)
	if !ok {
		return nil, fmt.Errorf("invalid URI argument: %#+v", args[0])
	}

	checker, ok := s.checkers[protocol.DocumentUri(uri)]
	if !ok {
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

	uri, ok := args[0].(string)
	if !ok {
		return nil, fmt.Errorf("invalid URI argument: %#+v", args[0])
	}

	checker, ok := s.checkers[protocol.DocumentUri(uri)]
	if !ok {
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
// There should be exactly 1 argument:
//   * the DocumentURI of the file to submit
//   * the array of arguments
func (s *Server) parseEntryPointArguments(_ protocol.Conn, args ...interface{}) (interface{}, error) {

	err := CheckCommandArgumentCount(args, 2)
	if err != nil {
		return nil, err
	}

	uri, ok := args[0].(string)
	if !ok {
		return nil, fmt.Errorf("invalid URI argument: %#+v", args[0])
	}

	checker, ok := s.checkers[protocol.DocumentUri(uri)]
	if !ok {
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

// convertError converts a checker error to a diagnostic.
func convertError(err convertibleError) protocol.Diagnostic {
	startPosition := err.StartPosition()
	endPosition := err.EndPosition()

	var message strings.Builder
	message.WriteString(err.Error())

	if secondaryError, ok := err.(errors.SecondaryError); ok {
		message.WriteString(". ")
		message.WriteString(secondaryError.SecondaryError())
	}

	return protocol.Diagnostic{
		Message:  message.String(),
		Severity: protocol.SeverityError,
		Range: protocol.Range{
			Start: protocol.Position{
				Line:      float64(startPosition.Line - 1),
				Character: float64(startPosition.Column),
			},
			End: protocol.Position{
				Line:      float64(endPosition.Line - 1),
				Character: float64(endPosition.Column + 1),
			},
		},
	}
}

// convertHint converts a checker error to a diagnostic.
func convertHint(hint sema.Hint) protocol.Diagnostic {
	startPosition := hint.StartPosition()
	endPosition := hint.EndPosition()

	return protocol.Diagnostic{
		Message: hint.Hint(),
		// protocol.SeverityHint doesn't look prominent enough in VS Code,
		// only the first character of the range is highlighted.
		Severity: protocol.SeverityInformation,
		Range: protocol.Range{
			Start: protocol.Position{
				Line:      float64(startPosition.Line - 1),
				Character: float64(startPosition.Column),
			},
			End: protocol.Position{
				Line:      float64(endPosition.Line - 1),
				Character: float64(endPosition.Column + 1),
			},
		},
	}
}
