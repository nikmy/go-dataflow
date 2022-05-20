package dataflow

import (
    "errors"
    "math/rand"
    "strconv"
    "sync/atomic"
    "testing"
    "time"
)

func TestFuture_Subscribe(t *testing.T) {
    a := 7
    x := 2
    ch := make(chan byte)
    f := MakeFuture(func() (int, error) {
        return x * 2, nil
    }).Subscribe(func(result Result) {
        a *= MakeValue[int](result)
        close(ch)
    })
    <-ch
    b, err := GetReadyResult[int](&f)

    if err != nil || b != 4 || a != 28 {
        t.Log(a)
        t.Fail()
    }
}

func TestFuture_Then(t *testing.T) {
    _ = MakeFuture(func() (any, error) {
        time.Sleep(2 * time.Second)
        return rand.Int(), nil
    }).Then(func(x any) (any, error) {
        return strconv.Itoa(x.(int) / 7), nil
    }).Then(func(_ any) (any, error) {
        return 0, errors.New("skip")
    }).Then(func(_ any) (any, error) {
        t.Log("Don't execute Then if result fails")
        t.Fail()
        return nil, nil
    })
    time.Sleep(5 * time.Second)
}

func TestFuture_Recover(t *testing.T) {
    recovered := make(chan bool)

    _ = MakeFuture(func() (any, error) {
        time.Sleep(2 * time.Second)
        return rand.Int(), nil
    }).Then(func(x any) (any, error) {
        return strconv.Itoa(x.(int) / 7), nil
    }).Then(func(_ any) (any, error) {
        return 0, errors.New("handle")
    }).Recover(func(_ error) {
        close(recovered)
    }).Then(func(_ any) (any, error) {
        t.Log("Don't execute Then after Recover")
        t.Fail()
        return nil, nil
    })
    <-recovered

    time.Sleep(time.Second * 3)
}

func TestCombine_FirstOf(t *testing.T) {
    var index int32
    waitFor := func(ms any) int32 {
        time.Sleep(time.Millisecond * ms.(time.Duration))
        return atomic.AddInt32(&index, 1)
    }

    waiters := make([]Future, 0)
    for i := 0; i < 100; i++ {
        waiters = append(waiters, MakeFuture(func() (int32, error) {
            d := time.Duration(rand.Int() % 500)
            return waitFor(d), nil
        }))
    }

    f := FirstOf(waiters...)

    r, _ := GetResult[int32](&f)
    if r != 1 {
        t.Fail()
    }

    time.Sleep(time.Second * 2)
}

func TestCombine_All(t *testing.T) {
    var index int32
    waitFor := func(ms any) int32 {
        time.Sleep(time.Millisecond * ms.(time.Duration))
        return atomic.AddInt32(&index, 1)
    }

    waiters := make([]Future, 0)
    for i := 0; i < 100; i++ {
        waiters = append(waiters, MakeFuture(func() (int32, error) {
            d := time.Duration(rand.Int() % 500)
            return waitFor(d), nil
        }))
    }

    f := All(waiters...)
    r, _ := GetResult[[]any](&f)

    s := int32(0)
    for _, x := range r {
        s += x.(int32)
    }

    if s != 5050 {
        t.Fail()
    }

    time.Sleep(time.Second * 2)
}
