package stream

import (
	"context"
	"encoding/json"
	"log"

	"github.com/quic-go/quic-go"
	"github.com/terrywh/devkit/entity"
)

type StreamHandlerFunc struct {
	fn func(ctx context.Context, ss *SessionStream)
}

func (shf StreamHandlerFunc) ServeStream(ctx context.Context, ss *SessionStream) {
	shf.fn(ctx, ss)
}

type ServeMux struct {
	handler map[string]StreamHandler
}

func NewServeMux() (mux *ServeMux) {
	mux = &ServeMux{
		handler: make(map[string]StreamHandler),
	}
	return mux
}

func (mux ServeMux) Handle(path string, handler StreamHandler) {
	mux.handler[path] = handler
}

func (mux ServeMux) HandleFunc(path string, fn func(ctx context.Context, ss *SessionStream)) {
	mux.handler[path] = StreamHandlerFunc{fn}
}

func (mux ServeMux) ServeStream(ctx context.Context, ss *SessionStream) {
	defer ss.Close()
	path, err := ss.r.ReadString(':')
	if err != nil {
		log.Print(err)
		return
	}
	path = path[:len(path)-1]

	if handler, found := mux.handler[path]; found {
		handler.ServeStream(ctx, ss)
	} else {
		log.Println("<ServeMux.ServeStream> handle not found for path: ", path)
		json.NewEncoder(ss).Encode(entity.HttpResponse{Error: entity.ErrHandlerNotFound})
		ss.s.CancelRead(quic.StreamErrorCode(entity.ErrSessionNotFound.Code))
	}
}
