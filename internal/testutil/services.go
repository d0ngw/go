//go:build integration

// Package testutil provisions isolated integration-test databases.
package testutil

import (
	"context"
	"crypto/rand"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"net"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/gomodule/redigo/redis"
)

//go:embed testdata/cache.sql
var CacheSchema string

// Database owns one generated test database and its administration connection.
type Database struct {
	Addr, User, Password, Name string
	DB                         *sql.DB
	admin                      *sql.DB
}

// Env returns a test setting, using fallback only when the variable is unset.
func Env(name, fallback string) string {
	if value, ok := os.LookupEnv(name); ok {
		return value
	}
	return fallback
}

// NewDatabase creates and seeds a unique database and drops it during Close.
// MYSQL_USER needs permission to create and drop these test databases.
func NewDatabase(suite, schema string) (*Database, error) {
	prefix := Env("MYSQL_DATABASE", "go_toolkit_test")
	if !regexp.MustCompile(`^[A-Za-z0-9_]{1,30}$`).MatchString(prefix) {
		return nil, errors.New("MYSQL_DATABASE must contain 1 to 30 letters, digits, or underscores")
	}
	db := &Database{
		Addr:     Env("MYSQL_ADDR", "127.0.0.1:33306"),
		User:     Env("MYSQL_USER", "root"),
		Password: Env("MYSQL_PASSWORD", "123456"),
		Name:     fmt.Sprintf("%s_%s_%x", prefix, suite, randBytes()),
	}
	cfg := mysql.NewConfig()
	cfg.User, cfg.Passwd, cfg.Net, cfg.Addr = db.User, db.Password, "tcp", db.Addr
	cfg.Timeout, cfg.ReadTimeout, cfg.WriteTimeout = 5*time.Second, 5*time.Second, 5*time.Second
	cfg.ParseTime = true
	admin, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		return nil, err
	}
	db.admin = admin
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := admin.PingContext(ctx); err != nil {
		admin.Close()
		return nil, fmt.Errorf("connect to disposable MySQL: %w", err)
	}
	if _, err := admin.ExecContext(ctx, "CREATE DATABASE `"+db.Name+"` CHARACTER SET utf8mb4"); err != nil {
		admin.Close()
		return nil, err
	}
	cfg.DBName = db.Name
	db.DB, err = sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		db.Close()
		return nil, err
	}
	for _, statement := range strings.Split(schema, ";") {
		if strings.TrimSpace(statement) == "" {
			continue
		}
		if _, err := db.DB.ExecContext(ctx, statement); err != nil {
			db.Close()
			return nil, fmt.Errorf("seed test schema: %w", err)
		}
	}
	return db, nil
}

func randBytes() []byte {
	var b [6]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	return b[:]
}

// UniquePrefix isolates a package's Redis keys from other tests and user data.
func UniquePrefix() string { return fmt.Sprintf("go_toolkit_test_%x_", randBytes()) }

// CheckCleanup propagates cleanup errors to the package test exit status.
// Defer this function with a pointer to a named result in the TestMain helper.
func CheckCleanup(code *int, cleanup func() error) {
	if err := cleanup(); err != nil {
		fmt.Fprintln(os.Stderr, "integration cleanup:", err)
		*code = 1
	}
}

// CleanupRedis deletes only the current suite's generated keys.
func CleanupRedis(prefix string) error {
	if prefix == "" {
		return errors.New("refusing Redis cleanup with an empty test prefix")
	}
	options := []redis.DialOption{
		redis.DialConnectTimeout(5 * time.Second), redis.DialReadTimeout(5 * time.Second), redis.DialWriteTimeout(5 * time.Second),
	}
	if password := Env("REDIS_PASSWORD", ""); password != "" {
		options = append(options, redis.DialPassword(password))
	}
	conn, err := redis.Dial("tcp", Env("REDIS_ADDR", "127.0.0.1:36379"), options...)
	if err != nil {
		return err
	}
	defer conn.Close()
	var cursor int
	for {
		values, err := redis.Values(conn.Do("SCAN", cursor, "MATCH", prefix+"*", "COUNT", 100))
		if err != nil {
			return err
		}
		var keys []string
		if _, err := redis.Scan(values, &cursor, &keys); err != nil {
			return err
		}
		if len(keys) > 0 {
			args := make([]interface{}, len(keys))
			for i, key := range keys {
				args[i] = key
			}
			if _, err := conn.Do("DEL", args...); err != nil {
				return err
			}
		}
		if cursor == 0 {
			return nil
		}
	}
}

func (db *Database) Close() error {
	var failures []error
	if db.DB != nil {
		failures = append(failures, db.DB.Close())
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := db.admin.ExecContext(ctx, "DROP DATABASE `"+db.Name+"`")
	failures = append(failures, err, db.admin.Close())
	return errors.Join(failures...)
}

// RedisAddress resolves the explicitly configured disposable Redis server.
func RedisAddress() (host string, port int, err error) {
	host, rawPort, err := net.SplitHostPort(Env("REDIS_ADDR", "127.0.0.1:36379"))
	if err != nil {
		return "", 0, err
	}
	port, err = strconv.Atoi(rawPort)
	if err != nil || port < 1 || port > 65535 {
		return "", 0, errors.New("invalid REDIS_ADDR port")
	}
	return host, port, nil
}
