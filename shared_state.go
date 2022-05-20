package dataflow

import "sync/atomic"

type rendezvous struct {
    n int32
}

func (r rendezvous) Again() {
    atomic.StoreInt32(&r.n, 0)
}

func (r *rendezvous) Come() bool {
    return atomic.AddInt32(&r.n, 1) == 2
}

type Callback func(Result)

type sharedState struct {
    result   Result
    callback Callback
    rend     rendezvous
    ready    chan bool
}

func (s *sharedState) makeRendezvous() {
    if s.rend.Come() {
        s.callback(s.result)
    }
}

func (s *sharedState) SetCallback(callback func(Result)) {
    s.callback = callback
    s.makeRendezvous()
}

func (s *sharedState) SetResult(value Result) {
    s.result = value
    s.ready <- true
    s.makeRendezvous()
}

func (s *sharedState) Cancel(err error) {
    s.result.Set(nil, err)
    s.ready <- false
    s.makeRendezvous()
}
