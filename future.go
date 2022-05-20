package dataflow

/**
Future are representation of asynchronous operation result. It is convenient to use it for
dataflow declarative programming. Unlike channels, it is unnecessary to wait a result. Use
futures if you want guaranteed wait-freedom, i.e. you want to declare a transaction.
*/

/*
MakeContract
    Create Future with Promise
*/
func MakeContract() (Future, Promise) {
    s := &sharedState{
        result:   Result{nil, nil},
        callback: nil,
        rend:     rendezvous{0},
        ready:    make(chan bool, 1),
    }
    return Future{s}, Promise{s}
}

type Promise struct {
    s *sharedState
}

func (p *Promise) Keep(r Result) {
    if p.s == nil {
        panic("Invalid promise!")
    }
    p.s.SetResult(r)
    p.s = nil
}

func (p *Promise) Fail(err error) {
    if p.s == nil {
        panic("Invalid promise!")
    }
    p.s.Cancel(err)
    p.s = nil
}

/*
Future
    Representation of async computation
*/
type Future struct {
    s *sharedState
}

/*
MakeFuture
    Create Future as representation of computation result
*/
func MakeFuture[T any](compute func() (T, error)) Future {
    f, p := MakeContract()
    go func() {
        p.Keep(MakeResult(compute()))
    }()
    return f
}

/*
GetReadyResult
    Non-blocking, use it for get result from ready Future
*/
func GetReadyResult(f *Future) (any, error) {
    val, err := f.s.result.Unpack()
    f.s = nil
    return val, err
}

/*
GetResult
    Blocks until Future is ready.
    Do not use for futures with callback
*/
func GetResult(f *Future) (any, error) {
    <-f.s.ready
    return GetReadyResult(f)
}

/*
Subscribe
    Set callback to the Future
    Use Then, Recover and combinators instead if possible
*/
func (f Future) Subscribe(callback Callback) Future {
    f.s.SetCallback(callback)
    return f
}

/*
Then
    Synchronous: X -> Y(X)
    Continue if there is no error
*/
func (f Future) Then(continuation func(value any) (any, error)) Future {
    ff, p := MakeContract()
    f.Subscribe(func(input Result) {
        if input.IsOk() {
            p.Keep(MakeResult(continuation(input.Value)))
        } else {
            p.Fail(input.Err)
        }
    })
    return ff
}

/*
ThenAsync
    Asynchronous: X -> Future(X)
    Continue if there is no error
*/
func (f Future) ThenAsync(async func(value any) Future) Future {
    ff, p := MakeContract()
    f.Subscribe(func(input Result) {
        if input.IsOk() {
            async(input.Value).Subscribe(func(result Result) {
                p.Keep(result)
            })
        } else {
            p.Fail(input.Err)
        }
    })
    return ff
}

/*
Recover
    If obtained Result contains error, it will be handled, and new result will be set.
    If there are no errors, Result will be passed on.
*/
func (f Future) Recover(handler func(error) (any, error)) Future {
    ff, p := MakeContract()
    f.Subscribe(func(input Result) {
        if input.HasError() {
            p.Keep(MakeResult(handler(input.Err)))
        } else {
            p.Keep(input)
        }
    })
    return ff
}

/*
FirstOf
    Continues with first arrived value or first error
*/
func FirstOf(inputs ...Future) Future {
    f, p := MakeContract()
    results := make(chan Result, len(inputs))
    for _, input := range inputs {
        input.Subscribe(func(result Result) {
            results <- result
        })
    }
    go func() {
        result := <-results
        if result.IsOk() {
            p.Keep(result)
        } else {
            p.Fail(result.Err)
        }
    }()
    return f
}

/*
All
    Continues with slice of all arrived values or first error
*/
func All(inputs ...Future) Future {
    f, p := MakeContract()
    results, fail := make(chan any, len(inputs)), make(chan error, 1)
    for _, input := range inputs {
        input.Subscribe(func(result Result) {
            if result.IsOk() {
                results <- result.Value
            } else {
                fail <- result.Err
            }
        })
    }

    toWait := len(inputs)
    values := make([]any, 0)
    getResults := func() {
        for {
            select {
            case val := <-results:
                values = append(values, val)
                toWait -= 1
                if toWait == 0 {
                    p.Keep(MakeResult(values, nil))
                    return
                }
            case err := <-fail:
                // TODO: cancellation
                p.Fail(err)
                return
            }
        }
    }
    go getResults()
    return f
}
