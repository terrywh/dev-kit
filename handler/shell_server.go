package handler

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"log"
	"sync"

	"github.com/quic-go/quic-go"
	"github.com/terrywh/devkit/entity"
	"github.com/terrywh/devkit/infra"
	"github.com/terrywh/devkit/stream"
)

type ServerShellHandler struct {
	StreamHandlerBase
	start map[entity.ShellID]*ServerShell
	mutex *sync.RWMutex
}

type ServerShell struct {
	entity.StartShell
	cpty infra.Pseudo `json:"-"`
}

func NewServerShellHandler(mux *stream.ServeMux) (h *ServerShellHandler) {
	h = &ServerShellHandler{
		start: make(map[entity.ShellID]*ServerShell),
		mutex: &sync.RWMutex{},
	}
	mux.HandleFunc("/shell/start", h.HandleStart)
	mux.HandleFunc("/shell/resize", h.HandleResize)
	// TODO cleanup
	return h
}

func (h *ServerShellHandler) put(e *ServerShell) {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	h.start[e.ShellId] = e
}

func (h *ServerShellHandler) del(e *ServerShell) {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	delete(h.start, e.ShellId)
}

func (h *ServerShellHandler) get(id entity.ShellID) *ServerShell {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	return h.start[id]
}

func (hss *ServerShellHandler) HandleStart(ctx context.Context, r *bufio.Reader, w io.WriteCloser) {
	var err error
	e := &ServerShell{}
	json.NewDecoder(r).Decode(&e)
	e.ApplyDefaults()

	e.cpty, err = infra.StartPty(ctx, e.Rows, e.Cols, e.ShellCmd[0], e.ShellCmd[1:]...)
	if err != nil {
		log.Println("<HandlerServerShell.HandleStream> failed to open pty shell: ", err)
		hss.Failure(w, err)
		return
	}
	defer e.cpty.Close()

	hss.put(e)
	defer hss.del(e)

	log.Println("<ServerShellHandler> shell: ", &e.cpty, " started ...")
	wg := &sync.WaitGroup{}
	wg.Add(2)
	go func() {
		defer wg.Done()
		io.Copy(w, e.cpty)
		w.(quic.Stream).CancelRead(quic.StreamErrorCode(0)) // 当 cpty 已关闭，不再处理用户输入
		w.Close()
	}()
	go func() {
		defer wg.Done()
		io.Copy(e.cpty, r)
		e.cpty.Close()
	}()
	wg.Wait()
	log.Println("<ServerShellHandler> shell: ", &e.cpty, " closed.")
}

func (hss *ServerShellHandler) HandleResize(ctx context.Context, r *bufio.Reader, w io.WriteCloser) {
	e1 := &ServerShell{}
	json.NewDecoder(r).Decode(&e1)

	e2 := hss.get(e1.ShellId)
	if e2 == nil {
		hss.Failure(w, entity.ErrShellNotFound)
		return
	}
	e2.Cols = e1.Cols
	e2.Rows = e1.Rows
	if err := e2.cpty.Resize(e2.Cols, e2.Rows); err != nil {
		log.Println("<ServerShellHandler.HandleResize> failed to resize cpty: ", err)
		hss.Failure(w, err)
		return
	}
	hss.Success(w, nil)
}