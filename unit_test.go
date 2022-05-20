package dataflow

import (
    "errors"
    "math/rand"
    "strconv"
    "sync"
    "sync/atomic"
    "testing"
    "time"
)

func TestFuture_Subscribe(t *testing.T) {
    a := 7
    x := 2
    done := make(chan bool)
    f := MakeFuture(func() (int, error) {
        time.Sleep(time.Second)
        return x * 2, nil
    }).Subscribe(func(result Result) {
        a *= MakeValue[int](result)
        close(done)
    })
    if a != 7 {
        t.Fail()
    }
    <-done

    b, err := GetReadyResult[int](&f)
    if err != nil || b != 4 || a != 28 {
        t.Log(a)
        t.Fail()
    }
}

func TestFuture_Then(t *testing.T) {
    done := make(chan bool)
    _ = MakeFuture(func() (any, error) {
        time.Sleep(2 * time.Second)
        return rand.Int(), nil
    }).Then(func(x any) (any, error) {
        return strconv.Itoa(x.(int) / 7), nil
    }).Then(func(_ any) (any, error) {
        close(done)
        return 0, errors.New("skip")
    }).Then(func(_ any) (any, error) {
        t.Log("Don't execute Then if result fails")
        t.Fail()
        return nil, nil
    })
    <-done
    time.Sleep(time.Second * 3)
}

func TestFuture_ThenAsync(t *testing.T) {
    makeRequest := func(_ any) Future {
        return MakeFuture(func() (any, error) {
            time.Sleep(500 * time.Millisecond)
            return 42, nil
        })
    }

    done := make(chan bool)

    _ = MakeFuture(func() (any, error) {
        return rand.Int(), nil
    }).Then(func(x any) (any, error) {
        return strconv.Itoa(x.(int) / 7), nil
    }).ThenAsync(makeRequest).Then(func(x any) (any, error) {
        if x != 42 {
            t.Fail()
        }
        close(done)
        return nil, nil
    })
    <-done
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
    var wg sync.WaitGroup

    waitFor := func(ms any) int32 {
        time.Sleep(time.Millisecond * ms.(time.Duration))
        wg.Done()
        return atomic.AddInt32(&index, 1)
    }

    wg.Add(100)
    waiters := make([]Future, 0)
    for i := 0; i < 100; i++ {
        waiters = append(waiters, MakeFuture(func() (int32, error) {
            d := time.Duration(rand.Int() % 500)
            return waitFor(d), nil
        }))
    }

    f := FirstOf(waiters...)
    wg.Wait()

    r, _ := GetResult[int32](&f)

    if r != 1 {
        t.Fail()
    }
}

func TestCombine_All(t *testing.T) {
    var index int32
    var wg sync.WaitGroup

    waitFor := func(ms any) int32 {
        time.Sleep(time.Millisecond * ms.(time.Duration))
        wg.Done()
        return atomic.AddInt32(&index, 1)
    }

    wg.Add(100)
    waiters := make([]Future, 0)
    for i := 0; i < 100; i++ {
        waiters = append(waiters, MakeFuture(func() (int32, error) {
            d := time.Duration(rand.Int() % 500)
            return waitFor(d), nil
        }))
    }

    f := All(waiters...)
    wg.Wait()

    r, _ := GetResult[[]any](&f)

    s := int32(0)
    for _, x := range r {
        s += x.(int32)
    }

    if s != 5050 {
        t.Fail()
    }
}
