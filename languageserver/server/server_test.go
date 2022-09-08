package server

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/mock"

	"github.com/onflow/cadence/languageserver/protocol"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_ServerGetDiagnostics(t *testing.T) {

	srv, err := NewServer()
	require.NoError(t, err)

	code := `
	pub contract Foo {
		 init() {
		   self.bar()
		 }
	}
	`

	conn := &protocol.MockConn{}

	conn.On("LogMessage", mock.Anything)
	conn.On("PublishDiagnostics", mock.Anything).Return(nil)
	conn.On("Notify", mock.Anything, mock.Anything).Return(nil)

	_ = srv.DidOpenTextDocument(conn, &protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{
			URI:     "foo",
			Version: 1,
			Text:    code,
		},
	})

	diagnostics, err := srv.getDiagnostics(
		"foo",
		code,
		1,
		func(*protocol.LogMessageParams) {},
	)
	assert.Equal(t, len(diagnostics), 1)

	d := diagnostics[0]
	params := &protocol.CodeActionParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: "foo"},
		Context: protocol.CodeActionContext{
			Diagnostics: []protocol.Diagnostic{{
				Range:              d.Range,
				Severity:           d.Severity,
				Code:               d.Code,
				CodeDescription:    d.CodeDescription,
				Source:             d.Source,
				Message:            d.Message,
				Tags:               d.Tags,
				RelatedInformation: d.RelatedInformation,
				Data:               fmt.Sprintf("%s", d.Data), // for some reason I have to pass it as converted to string else the type check fails since it's uuid
			}},
		},
	}
	actions, err := srv.CodeAction(nil, params)

	assert.NoError(t, err)
	assert.Len(t, actions, 1)

	fmt.Println(actions)
}
