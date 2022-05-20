package dataflow

type Result struct {
    Value any
    Err   error
}

func MakeResult(value any, err error) Result {
    return Result{value, err}
}

func MakeValue[T any](result Result) T {
    return result.Value.(T)
}

func (r *Result) HasValue() bool {
    return r.Value != nil
}

func (r *Result) HasError() bool {
    return r.Err != nil
}

func (r *Result) IsOk() bool {
    return !r.HasError()
}

func (r *Result) Set(value any, err error) {
    if value != nil {
        r.Value = value
    }
    if err != nil {
        r.Err = err
    }
}

func (r Result) Unpack() (any, error) {
    return r.Value, r.Err
}
