package wfcache

import (
	"context"
)

type Future interface {
	Await() (interface{}, error)
	AwaitWithContext(context.Context) (interface{}, error)
}

type promise struct {
	await func(ctx context.Context) (interface{}, error)
}

func (f promise) Await() (interface{}, error) {
	return f.await(context.Background())
}

func (f promise) AwaitWithContext(ctx context.Context) (interface{}, error) {
	return f.await(ctx)
}

func Promise(f func() (interface{}, error)) Future {
	var val interface{}
	var err error

	c := make(chan struct{})
	go func() {
		defer close(c)
		val, err = f()
	}()

	return promise{
		await: func(ctx context.Context) (interface{}, error) {
			select {
			case <-c:
				if err != nil {
					return nil, err
				}
				return val, nil
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		},
	}
}
