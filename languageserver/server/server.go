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
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/onflow/cadence/languageserver/conversion"
	"github.com/onflow/cadence/languageserver/jsonrpc2"
	"github.com/onflow/cadence/runtime"
	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/parser2"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/stdlib"

	"github.com/onflow/cadence/languageserver/protocol"
)

var valueDeclarations = append(
	stdlib.FlowBuiltInFunctions(stdlib.FlowBuiltinImpls{}),
	stdlib.BuiltinFunctions...,
).ToValueDeclarations()
var typeDeclarations = append(
	stdlib.FlowBuiltInTypes,
	stdlib.BuiltinTypes...,
).ToTypeDeclarations()

// document represents an open document on the client. It contains all cached
// information about each document that is used to support CodeLens,
// transaction submission, and script execution.
type Document struct {
	Text          string
	latestVersion float64
	hasErrors     bool
}

// CommandHandler represents the form of functions that handle commands
// submitted from the client using workspace/executeCommand.
type CommandHandler func(conn protocol.Conn, args ...interface{}) (interface{}, error)

// AddressImportResolver is a function that is used to resolve address imports
//
type AddressImportResolver func(location ast.AddressLocation) (string, error)

// StringImportResolver is a function that is used to resolve string imports
//
type StringImportResolver func(mainPath string, location ast.StringLocation) (string, error)

// CodeLensProvider is a function that is used to provide code lenses for the given checker
//
type CodeLensProvider func(uri protocol.DocumentUri, checker *sema.Checker) ([]*protocol.CodeLens, error)

// DiagnosticProvider is a function that is used to provide diagnostics for the given checker
//
type DiagnosticProvider func(uri protocol.DocumentUri, checker *sema.Checker) ([]protocol.Diagnostic, error)

// InitializationOptionsHandler is a function that is used to handle initialization options sent by the client
//
type InitializationOptionsHandler func(initializationOptions interface{}) error

type Server struct {
	checkers  map[protocol.DocumentUri]*sema.Checker
	documents map[protocol.DocumentUri]Document
	// commands is the registry of custom commands we support
	commands map[string]CommandHandler
	// resolveAddressImport is the optional function that is used to resolve address imports
	resolveAddressImport AddressImportResolver
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

func NewServer() *Server {
	return &Server{
		checkers:  make(map[protocol.DocumentUri]*sema.Checker),
		documents: make(map[protocol.DocumentUri]Document),
		commands:  make(map[string]CommandHandler),
	}
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

func (s *Server) Start(stream jsonrpc2.ObjectStream) {
	<-protocol.NewServer(s).Start(stream)
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
			// TODO:
			//SignatureHelpProvider: &protocol.SignatureHelpOptions{
			//	TriggerCharacters: []string{"("},
			//},
			CodeLensProvider: &protocol.CodeLensOptions{
				ResolveProvider: false,
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
	conn.LogMessage(&protocol.LogMessageParams{
		Type:    protocol.Log,
		Message: "DidOpenTextDoc",
	})

	uri := params.TextDocument.URI
	text := params.TextDocument.Text

	diagnostics, err := s.getDiagnostics(conn, uri, text)
	if err != nil {
		return err
	}
	conn.PublishDiagnostics(&protocol.PublishDiagnosticsParams{
		URI:         uri,
		Diagnostics: diagnostics,
	})

	s.documents[uri] = Document{
		Text:          text,
		latestVersion: params.TextDocument.Version,
		hasErrors:     len(diagnostics) > 0,
	}

	return nil
}

// DidChangeTextDocument is called whenever the current document changes.
// We parse and check the text and publish diagnostics about the document.
func (s *Server) DidChangeTextDocument(
	conn protocol.Conn,
	params *protocol.DidChangeTextDocumentParams,
) error {
	conn.LogMessage(&protocol.LogMessageParams{
		Type:    protocol.Log,
		Message: fmt.Sprintf("DidChangeText changes: %d", len(params.ContentChanges)),
	})
	uri := params.TextDocument.URI
	text := params.ContentChanges[0].Text

	diagnostics, err := s.getDiagnostics(conn, uri, text)
	if err != nil {
		return err
	}
	conn.PublishDiagnostics(&protocol.PublishDiagnosticsParams{
		URI:         uri,
		Diagnostics: diagnostics,
	})

	s.documents[uri] = Document{
		Text:          text,
		latestVersion: params.TextDocument.Version,
		hasErrors:     len(diagnostics) > 0,
	}

	return nil
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
		Value: fmt.Sprintf("* Type: `%s`", occurrence.Origin.Type.QualifiedString()),
	}
	return &protocol.Hover{Contents: contents}, nil
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
func (s *Server) CodeLens(_ protocol.Conn, params *protocol.CodeLensParams) ([]*protocol.CodeLens, error) {

	uri := params.TextDocument.URI
	checker, ok := s.checkers[uri]
	if !ok {
		// Can we ensure this doesn't happen?
		return []*protocol.CodeLens{}, nil
	}

	var allCodeLenses []*protocol.CodeLens

	for _, provider := range s.codeLensProviders {
		moreCodeLenses, err := provider(uri, checker)
		if err != nil {
			return nil, err
		}

		allCodeLenses = append(allCodeLenses, moreCodeLenses...)
	}

	return allCodeLenses, nil
}

// ExecuteCommand is called to execute a custom, server-defined command.
//
// We register all the commands we support in registerCommands and populate
// their corresponding handler at server initialization.
func (s *Server) ExecuteCommand(conn protocol.Conn, params *protocol.ExecuteCommandParams) (interface{}, error) {
	conn.LogMessage(&protocol.LogMessageParams{
		Type:    protocol.Log,
		Message: "called execute command: " + params.Command,
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
func (s *Server) getDiagnostics(conn protocol.Conn, uri protocol.DocumentUri, text string) ([]protocol.Diagnostic, error) {
	diagnostics := make([]protocol.Diagnostic, 0)
	program, err := parse(conn, text, string(uri))

	// If there were parsing errors, convert each one to a diagnostic and exit
	// without checking.
	if err != nil {
		if parentErr, ok := err.(errors.ParentError); ok {
			parserDiagnostics := getDiagnosticsForParentError(conn, parentErr)
			diagnostics = append(diagnostics, parserDiagnostics...)
			return diagnostics, nil
		}
		return nil, err
	}

	// There were no parser errors. Proceed to resolving imports and
	// checking the parsed program.

	mainPath := string(uri)

	if strings.HasPrefix(mainPath, filePrefix) {
		mainPath = mainPath[len(filePrefix):]
	} else if strings.HasPrefix(mainPath, inMemoryPrefix) {
		mainPath = mainPath[len(inMemoryPrefix):]
	}

	_ = program.ResolveImports(func(location ast.Location) (program *ast.Program, err error) {
		return s.resolveImport(conn, mainPath, location)
	})

	checker, err := sema.NewChecker(
		program,
		runtime.FileLocation(uri),
		sema.WithPredeclaredValues(valueDeclarations),
		sema.WithPredeclaredTypes(typeDeclarations),
	)
	if err != nil {
		return nil, err
	}

	start := time.Now()
	err = checker.Check()
	elapsed := time.Since(start)

	// Log how long it took to check the file
	conn.LogMessage(&protocol.LogMessageParams{
		Type:    protocol.Info,
		Message: fmt.Sprintf("checking %s took %s", string(uri), elapsed),
	})

	s.checkers[uri] = checker

	if err != nil {
		if parentErr, ok := err.(errors.ParentError); ok {
			checkerDiagnostics := getDiagnosticsForParentError(conn, parentErr)
			diagnostics = append(diagnostics, checkerDiagnostics...)
		} else {
			return nil, err
		}
	}

	for _, provider := range s.diagnosticProviders {
		var extraDiagnostics []protocol.Diagnostic
		extraDiagnostics, err = provider(uri, checker)
		if err != nil {
			return nil, err
		}
		diagnostics = append(diagnostics, extraDiagnostics...)
	}

	return diagnostics, nil
}

// getDiagnosticsForParentError unpacks all child errors and converts each to
// a diagnostic. Both parser and checker errors can be unpacked.
//
// Logs any conversion failures to the client.
func getDiagnosticsForParentError(conn protocol.Conn, err errors.ParentError) (diagnostics []protocol.Diagnostic) {
	for _, childErr := range err.ChildErrors() {
		convertibleErr, ok := childErr.(convertibleError)
		if !ok {
			conn.LogMessage(&protocol.LogMessageParams{
				Type:    protocol.Error,
				Message: fmt.Sprintf("Unable to convert non-convertable error to disgnostic: %s", err.Error()),
			})
			continue
		}
		diagnostic := convertError(convertibleErr)
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

func (s *Server) resolveImport(
	_ protocol.Conn,
	mainPath string,
	location ast.Location,
) (
	program *ast.Program,
	err error,
) {
	var code string
	switch loc := location.(type) {
	case ast.StringLocation:
		if s.resolveStringImport == nil {
			return nil, fmt.Errorf("unable to resolve string location %s", loc)
		}
		code, err = s.resolveStringImport(mainPath, loc)
	case ast.AddressLocation:
		if s.resolveAddressImport == nil {
			return nil, fmt.Errorf("unable to resolve string location %s", loc)
		}
		code, err = s.resolveAddressImport(loc)
	default:
		return nil, fmt.Errorf("unable to resolve address location %s", loc)
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
