package pool

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/o3willard-AI/SSSonector/internal/config/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockConn struct {
	net.Conn
	closed bool
}

func (m *mockConn) Close() error {
	m.closed = true
	return nil
}

func TestPool(t *testing.T) {
	t.Run("basic pool operations", func(t *testing.T) {
		factory := func(ctx context.Context) (net.Conn, error) {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
				return &mockConn{}, nil
			}
		}

		config := Config{
			InitialSize:    1,
			MaxSize:        2,
			MinSize:        1,
			MaxIdleTime:    types.NewDuration(time.Minute),
			ConnectTimeout: types.NewDuration(time.Second),
			Factory:        factory,
		}

		pool, err := NewPool(config)
		require.NoError(t, err)
		defer pool.Close()

		// Get a connection
		conn1, err := pool.Get(context.Background())
		require.NoError(t, err)
		require.NotNil(t, conn1)

		// Return it to the pool
		err = pool.Put(conn1)
		require.NoError(t, err)

		// Get it again
		conn2, err := pool.Get(context.Background())
		require.NoError(t, err)
		require.NotNil(t, conn2)
	})

	t.Run("pool exhaustion", func(t *testing.T) {
		factory := func(ctx context.Context) (net.Conn, error) {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
				return &mockConn{}, nil
			}
		}

		config := Config{
			InitialSize:    1,
			MaxSize:        1,
			MinSize:        1,
			MaxIdleTime:    types.NewDuration(time.Minute),
			ConnectTimeout: types.NewDuration(time.Second),
			Factory:        factory,
		}

		pool, err := NewPool(config)
		require.NoError(t, err)
		defer pool.Close()

		// Get the only connection
		conn1, err := pool.Get(context.Background())
		require.NoError(t, err)
		require.NotNil(t, conn1)

		// Try to get another connection with a timeout
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		// Wait for the context to expire
		time.Sleep(15 * time.Millisecond)

		_, err = pool.Get(ctx)
		assert.ErrorIs(t, err, context.DeadlineExceeded)
	})

	t.Run("health check failure", func(t *testing.T) {
		failingHealthCheck := func(conn net.Conn) error {
			return errors.New("health check failed")
		}

		factory := func(ctx context.Context) (net.Conn, error) {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
				return &mockConn{}, nil
			}
		}

		config := Config{
			InitialSize:    1,
			MaxSize:        2,
			MinSize:        1,
			MaxIdleTime:    types.NewDuration(time.Minute),
			ConnectTimeout: types.NewDuration(time.Second),
			Factory:        factory,
			HealthCheck:    failingHealthCheck,
		}

		pool, err := NewPool(config)
		require.NoError(t, err)
		defer pool.Close()

		// Get a connection - should create a new one since health check fails
		conn1, err := pool.Get(context.Background())
		require.NoError(t, err)
		require.NotNil(t, conn1)
	})

	t.Run("connection timeout", func(t *testing.T) {
		factory := func(ctx context.Context) (net.Conn, error) {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(50 * time.Millisecond):
				return &mockConn{}, nil
			}
		}

		config := Config{
			InitialSize:    0,
			MaxSize:        1,
			MinSize:        0,
			MaxIdleTime:    types.NewDuration(time.Minute),
			ConnectTimeout: types.NewDuration(time.Second),
			Factory:        factory,
		}

		pool, err := NewPool(config)
		require.NoError(t, err)
		defer pool.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		_, err = pool.Get(ctx)
		assert.ErrorIs(t, err, context.DeadlineExceeded)
	})
}
