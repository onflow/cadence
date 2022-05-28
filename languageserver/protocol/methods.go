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

package protocol

import "encoding/json"

func (s *Server) handleInitialize(req *json.RawMessage) (interface{}, error) {
	var params InitializeParams
	if err := json.Unmarshal(*req, &params); err != nil {
		return nil, err
	}

	return s.Handler.Initialize(s.conn, &params)
}

func (s *Server) handleDidOpenTextDocument(req *json.RawMessage) (interface{}, error) {
	var params DidOpenTextDocumentParams
	if err := json.Unmarshal(*req, &params); err != nil {
		return nil, err
	}

	err := s.Handler.DidOpenTextDocument(s.conn, &params)
	return nil, err
}

func (s *Server) handleDidChangeTextDocument(req *json.RawMessage) (interface{}, error) {
	var params DidChangeTextDocumentParams
	if err := json.Unmarshal(*req, &params); err != nil {
		return nil, err
	}

	err := s.Handler.DidChangeTextDocument(s.conn, &params)
	return nil, err
}

func (s *Server) handleHover(req *json.RawMessage) (interface{}, error) {
	var params TextDocumentPositionParams
	if err := json.Unmarshal(*req, &params); err != nil {
		return nil, err
	}

	return s.Handler.Hover(s.conn, &params)
}

func (s *Server) handleDefinition(req *json.RawMessage) (interface{}, error) {
	var params TextDocumentPositionParams
	if err := json.Unmarshal(*req, &params); err != nil {
		return nil, err
	}

	return s.Handler.Definition(s.conn, &params)
}

func (s *Server) handleSignatureHelp(req *json.RawMessage) (interface{}, error) {
	var params TextDocumentPositionParams
	if err := json.Unmarshal(*req, &params); err != nil {
		return nil, err
	}

	return s.Handler.SignatureHelp(s.conn, &params)
}

func (s *Server) handleDocumentHighlight(req *json.RawMessage) (interface{}, error) {
	var params TextDocumentPositionParams
	if err := json.Unmarshal(*req, &params); err != nil {
		return nil, err
	}

	return s.Handler.DocumentHighlight(s.conn, &params)
}

func (s *Server) handleRename(req *json.RawMessage) (interface{}, error) {
	var params RenameParams
	if err := json.Unmarshal(*req, &params); err != nil {
		return nil, err
	}

	return s.Handler.Rename(s.conn, &params)
}

func (s *Server) handleCodeAction(req *json.RawMessage) (interface{}, error) {
	var params CodeActionParams
	if err := json.Unmarshal(*req, &params); err != nil {
		return nil, err
	}

	return s.Handler.CodeAction(s.conn, &params)
}

func (s *Server) handleCodeLens(req *json.RawMessage) (interface{}, error) {
	var params CodeLensParams
	if err := json.Unmarshal(*req, &params); err != nil {
		return nil, err
	}

	return s.Handler.CodeLens(s.conn, &params)
}

func (s *Server) handleCompletion(req *json.RawMessage) (interface{}, error) {
	var params CompletionParams
	if err := json.Unmarshal(*req, &params); err != nil {
		return nil, err
	}

	return s.Handler.Completion(s.conn, &params)
}

func (s *Server) handleCompletionItemResolve(req *json.RawMessage) (interface{}, error) {
	var completionItem CompletionItem
	if err := json.Unmarshal(*req, &completionItem); err != nil {
		return nil, err
	}

	return s.Handler.ResolveCompletionItem(s.conn, &completionItem)
}

func (s *Server) handleExecuteCommand(req *json.RawMessage) (interface{}, error) {
	var params ExecuteCommandParams
	if err := json.Unmarshal(*req, &params); err != nil {
		return nil, err
	}

	return s.Handler.ExecuteCommand(s.conn, &params)
}

func (s *Server) handleDocumentSymbol(req *json.RawMessage) (interface{}, error) {
	var params DocumentSymbolParams
	if err := json.Unmarshal(*req, &params); err != nil {
		return nil, err
	}
	return s.Handler.DocumentSymbol(s.conn, &params)
}

func (s *Server) handleShutdown(_ *json.RawMessage) (interface{}, error) {
	err := s.Handler.Shutdown(s.conn)
	return nil, err
}

func (s *Server) handleExit(_ *json.RawMessage) (interface{}, error) {
	err := s.Handler.Exit(s.conn)
	return nil, err
}
