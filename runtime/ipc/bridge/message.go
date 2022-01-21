package bridge

type messageKind int8

const (
	REQUEST messageKind = iota
	RESPONSE
	ERROR
	DONE
)

type Message interface {
	IsMessage()
	String() string
}

var _ Message = &Request{}
var _ Message = &Response{}
var _ Message = &Error{}

type Request struct {
	Name   string
	Params []interface{}
}

func (Request) IsMessage() {}

func (r *Request) String() string {
	return r.Name
}

type Response struct {
	Content string
}

func (Response) IsMessage() {}

func (r *Response) String() string {
	return r.Content
}

type Error struct {
	Content string
}

func (Error) IsMessage() {}

func (e *Error) String() string {
	return e.Content
}
