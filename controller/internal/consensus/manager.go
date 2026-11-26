package consensus

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/concurrency"
)

var (
	ErrNotLeader  = errors.New("leadership lost or fencing token rejected")
	ErrInvalidKey = errors.New("invalid protected key")
)

const (
	DefaultTTL      = 10
	DefaultLockKey  = "/atlas/leadership/lock"
	DefaultFenceKey = "/atlas/fencing/last-token"
)

type Config struct {
	TTL        int
	LockKey    string
	FencingKey string
}

type Manager struct {
	client     *clientv3.Client
	ttl        int
	lockKey    string
	fencingKey string
}

type Leadership struct {
	manager *Manager
	session *concurrency.Session
	mutex   *concurrency.Mutex
	token   int64

	releaseOnce sync.Once
	releaseErr  error
}

func normalizeConfig(cfg Config) (Config, error) {
	if cfg.TTL <= 0 {
		cfg.TTL = DefaultTTL
	}
	if cfg.LockKey == "" {
		cfg.LockKey = DefaultLockKey
	}
	if cfg.FencingKey == "" {
		cfg.FencingKey = DefaultFenceKey
	}
	if cfg.LockKey == cfg.FencingKey {
		return Config{}, errors.New("lock key and fencing key must differ")
	}
	return cfg, nil
}

func NewManager(client *clientv3.Client, cfg Config) (*Manager, error) {
	if client == nil {
		return nil, errors.New("etcd client cannot be nil")
	}

	cfg, err := normalizeConfig(cfg)
	if err != nil {
		return nil, err
	}

	return &Manager{
		client:     client,
		ttl:        cfg.TTL,
		lockKey:    cfg.LockKey,
		fencingKey: cfg.FencingKey,
	}, nil
}

func (m *Manager) Acquire(ctx context.Context) (*Leadership, error) {
	session, err := concurrency.NewSession(
		m.client,
		concurrency.WithTTL(m.ttl),
	)
	if err != nil {
		return nil, fmt.Errorf("create etcd session: %w", err)
	}

	mutex := concurrency.NewMutex(session, m.lockKey)

	if err := mutex.Lock(ctx); err != nil {
		_ = session.Close()
		return nil, fmt.Errorf("acquire leadership: %w", err)
	}

	header := mutex.Header()
	if header == nil || header.Revision <= 0 {
		_ = mutex.Unlock(context.Background())
		_ = session.Close()
		return nil, errors.New("etcd returned an invalid leadership revision")
	}

	token := header.Revision

	if err := registerFencingToken(
		ctx,
		m.client,
		mutex,
		m.fencingKey,
		token,
	); err != nil {
		_ = mutex.Unlock(context.Background())
		_ = session.Close()
		return nil, fmt.Errorf("register fencing token: %w", err)
	}

	return &Leadership{
		manager: m,
		session: session,
		mutex:   mutex,
		token:   token,
	}, nil
}

func registerFencingToken(
	ctx context.Context,
	client *clientv3.Client,
	mutex *concurrency.Mutex,
	fencingKey string,
	token int64,
) error {
	// Initialize the fencing key once, while holding the leadership lock.
	_, err := client.Txn(ctx).
		If(
			mutex.IsOwner(),
			clientv3.Compare(clientv3.Version(fencingKey), "=", 0),
		).
		Then(
			clientv3.OpPut(fencingKey, formatToken(0)),
		).
		Commit()
	if err != nil {
		return fmt.Errorf("initialize fencing key: %w", err)
	}

	fencingValue := formatToken(token)

	// Register this acquisition's token exactly once.
	response, err := client.Txn(ctx).
		If(
			mutex.IsOwner(),
			clientv3.Compare(
				clientv3.Value(fencingKey),
				"<",
				fencingValue,
			),
		).
		Then(
			clientv3.OpPut(fencingKey, fencingValue),
		).
		Commit()
	if err != nil {
		return fmt.Errorf("register token: %w", err)
	}
	if !response.Succeeded {
		return ErrNotLeader
	}

	return nil
}

func (l *Leadership) Token() int64 {
	return l.token
}

// LeaseID exposes the session lease ID for diagnostics and controlled
// failure experiments. It is not an authorization credential.
func (l *Leadership) LeaseID() int64 {
	return int64(l.session.Lease())
}

func (l *Leadership) Done() <-chan struct{} {
	return l.session.Done()
}

func (l *Leadership) IsLeader(ctx context.Context) (bool, error) {
	select {
	case <-l.session.Done():
		return false, nil
	default:
	}

	response, err := l.manager.client.Txn(ctx).
		If(l.mutex.IsOwner()).
		Then(clientv3.OpGet(l.mutex.Key())).
		Commit()

	if err != nil {
		return false, fmt.Errorf("check leadership ownership: %w", err)
	}

	return response.Succeeded, nil
}

func validateProtectedKey(key, lockKey, fencingKey string) error {
	if key == "" ||
		key == lockKey ||
		strings.HasPrefix(key, lockKey+"/") ||
		key == fencingKey ||
		strings.HasPrefix(key, fencingKey+"/") {
		return ErrInvalidKey
	}
	return nil
}

func (l *Leadership) FencedPut(
	ctx context.Context,
	key string,
	value string,
) error {
	if err := validateProtectedKey(
		key,
		l.manager.lockKey,
		l.manager.fencingKey,
	); err != nil {
		return err
	}

	// A leader may perform multiple writes using the same token.
	// The token must still be current and the mutex must still be owned.
	fencingValue := formatToken(l.token)

	response, err := l.manager.client.Txn(ctx).
		If(
			l.mutex.IsOwner(),
			clientv3.Compare(
				clientv3.Value(l.manager.fencingKey),
				"=",
				fencingValue,
			),
		).
		Then(
			clientv3.OpPut(key, value),
		).
		Commit()

	if err != nil {
		return fmt.Errorf("execute fenced write: %w", err)
	}

	if !response.Succeeded {
		return ErrNotLeader
	}

	return nil
}

func (l *Leadership) Release(ctx context.Context) error {
	l.releaseOnce.Do(func() {
		unlockErr := l.mutex.Unlock(ctx)
		closeErr := l.session.Close()
		l.releaseErr = errors.Join(unlockErr, closeErr)
	})
	return l.releaseErr
}

func formatToken(token int64) string {
	return fmt.Sprintf("%020d", token)
}
