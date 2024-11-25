// +build go1.4

package log

//import "sync/atomic"

type swapHandler struct {
	//handler atomic.Value
	handlers []Handler
}

func (h *swapHandler) Log(r *Record) error {
	for _,handler := range h.handlers {
		if handler != nil {
			handler.Log(r)
		}
	}
	return nil
	//return (*h.handler.Load().(*Handler)).Log(r)
}

func (h *swapHandler) Swap(newHandlers []Handler) {
	h.handlers = newHandlers
	//h.handler.Store(&newHandler)
}

func (h *swapHandler) Get() []Handler {
	return h.handlers
	//return *h.handler.Load().(*Handler)
}
