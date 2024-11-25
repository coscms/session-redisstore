package redisstore

import (
	"log"
	"strings"
	"time"

	"github.com/admpub/redistore"
	"github.com/admpub/securecookie"
	"github.com/admpub/sessions"
	ss "github.com/webx-top/echo/middleware/session/engine"
)

var DefaultMaxReconnect = 5

func New(opts *RedisOptions) sessions.Store {
	store, err := NewRedisStore(opts)
	if err != nil {
		if !strings.Contains(err.Error(), `connect:`) {
			panic(err.Error())
		}
		retries := opts.MaxReconnect
		if retries <= 0 {
			retries = DefaultMaxReconnect
		}
		for i := 1; i < retries; i++ {
			log.Println(`[sessions]`, err.Error())
			wait := time.Second
			log.Printf(`[sessions] (%d/%d) reconnect redis after %v`, i, retries, wait)
			time.Sleep(wait)
			store, err = NewRedisStore(opts)
			if err == nil {
				log.Println(`[sessions] reconnect redis successfully`)
				return store
			}
		}
		panic(err.Error())
	}
	return store
}

func Reg(store sessions.Store, args ...string) {
	name := `redis`
	if len(args) > 0 {
		name = args[0]
	}
	ss.Reg(name, store)
}

func RegWithOptions(opts *RedisOptions, args ...string) sessions.Store {
	store := New(opts)
	Reg(store, args...)
	return store
}

type RedisOptions struct {
	Size         int      `json:"size"`
	Network      string   `json:"network"`
	Address      string   `json:"address"`
	Password     string   `json:"password"`
	DB           uint     `json:"db"`
	KeyPairs     [][]byte `json:"keyPairs"`
	MaxAge       int      `json:"maxAge"`
	EmptyDataAge int      `json:"emptyDataAge"`
	MaxLength    int      `json:"maxLength"`
	MaxReconnect int      `json:"maxReconnect"`
}

// size: maximum number of idle connections.
// network: tcp or udp
// address: host:port
// password: redis-password
// Keys are defined in pairs to allow key rotation, but the common case is to set a single
// authentication key and optionally an encryption key.
//
// The first key in a pair is used for authentication and the second for encryption. The
// encryption key can be set to nil or omitted in the last pair, but the authentication key
// is required in all pairs.
//
// It is recommended to use an authentication key with 32 or 64 bytes. The encryption key,
// if set, must be either 16, 24, or 32 bytes to select AES-128, AES-192, or AES-256 modes.
func NewRedisStore(opts *RedisOptions) (sessions.Store, error) {
	store, err := redistore.NewRediStoreWithDB(opts.Size, opts.Network, opts.Address, opts.Password, int(opts.DB), opts.KeyPairs...)
	if err != nil {
		return nil, err
	}
	if opts.MaxAge > 0 {
		store.DefaultMaxAge = opts.MaxAge
	} else {
		store.DefaultMaxAge = ss.DefaultMaxAge
	}
	if opts.EmptyDataAge > 0 {
		store.EmptyDataAge = opts.EmptyDataAge
	} else {
		store.EmptyDataAge = ss.EmptyDataAge
	}
	s := &redisStore{store}
	if opts.MaxLength > 0 {
		s.MaxLength(opts.MaxLength)
	}
	return s, nil
}

type redisStore struct {
	*redistore.RediStore
}

// MaxLength restricts the maximum length of new sessions to l.
// If l is 0 there is no limit to the size of a session, use with caution.
// The default for a new FilesystemStore is 4096.
func (s *redisStore) MaxLength(l int) {
	securecookie.SetMaxLength(s.Codecs, l)
}
