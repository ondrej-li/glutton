package saver

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/defectus/glutton/pkg/iface"
	"github.com/stretchr/testify/assert"
)

const stubDriverName = "glutton-stub"

type stubDriver struct {
	mu       sync.Mutex
	pingErr  error
	execErr  error
	execSQL  string
	lastArgs []driver.NamedValue
}

func (d *stubDriver) Open(string) (driver.Conn, error) { return &stubConn{driver: d}, nil }

type stubConn struct {
	driver *stubDriver
}

func (c *stubConn) Prepare(string) (driver.Stmt, error) { return nil, errors.New("not implemented") }
func (c *stubConn) Close() error                        { return nil }
func (c *stubConn) Begin() (driver.Tx, error)           { return nil, errors.New("not implemented") }
func (c *stubConn) Ping(context.Context) error          { return c.driver.pingErr }

func (c *stubConn) ExecContext(_ context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	c.driver.mu.Lock()
	defer c.driver.mu.Unlock()
	c.driver.execSQL = query
	c.driver.lastArgs = args
	if c.driver.execErr != nil {
		return nil, c.driver.execErr
	}
	return driver.RowsAffected(1), nil
}

var (
	stub           = new(stubDriver)
	registerDriver sync.Once
)

func configureSaver(t *testing.T, layout string) *DatabaseSaver {
	registerDriver.Do(func() { sql.Register(stubDriverName, stub) })
	ds := new(DatabaseSaver)
	assert.NoError(t, ds.Configure(&iface.Settings{
		SQLDriver:           stubDriverName,
		SQLConnectionString: "stub",
		SQLayout:            layout,
	}))
	return ds
}

func TestDatabaseSaverConfigureUsesDefaultLayout(t *testing.T) {
	stub.pingErr = nil
	ds := configureSaver(t, "")
	defer ds.Close()
	assert.Equal(t, defaultLayout, ds.layout)
}

func TestDatabaseSaverConfigureFailsWhenUnreachable(t *testing.T) {
	registerDriver.Do(func() { sql.Register(stubDriverName, stub) })
	stub.pingErr = errors.New("unreachable")
	defer func() { stub.pingErr = nil }()

	ds := new(DatabaseSaver)
	err := ds.Configure(&iface.Settings{SQLDriver: stubDriverName, SQLConnectionString: "stub"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error connecting to database")
}

func TestDatabaseSaverSaveSerializesMeta(t *testing.T) {
	stub.pingErr = nil
	stub.execErr = nil
	ds := configureSaver(t, "INSERT INTO payload(ts, remote, meta, payload) VALUES ($1, $2, $3, $4)")
	defer ds.Close()

	meta := map[string][]string{"Content-Type": {"application/json"}}
	err := ds.Save(&iface.PayloadRecord{
		Payload:   "body",
		Timestamp: time.Now(),
		Remote:    "127.0.0.1",
		Meta:      meta,
	})
	assert.NoError(t, err)
	assert.Equal(t, "INSERT INTO payload(ts, remote, meta, payload) VALUES ($1, $2, $3, $4)", stub.execSQL)
	assert.Len(t, stub.lastArgs, 4)

	serialized, ok := stub.lastArgs[2].Value.(string)
	assert.True(t, ok, "meta must be passed as a string")
	var got map[string][]string
	assert.NoError(t, json.Unmarshal([]byte(serialized), &got))
	assert.Equal(t, meta, got)
}

func TestDatabaseSaverSaveReturnsError(t *testing.T) {
	stub.pingErr = nil
	stub.execErr = errors.New("exec failed")
	defer func() { stub.execErr = nil }()
	ds := configureSaver(t, "")
	defer ds.Close()

	err := ds.Save(&iface.PayloadRecord{Meta: map[string][]string{}})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error saving payload")
}

func TestDatabaseSaverCloseWithoutConnection(t *testing.T) {
	ds := new(DatabaseSaver)
	assert.NoError(t, ds.Close())
}

func TestDatabaseSaverPostgresIntegration(t *testing.T) {
	dsn := os.Getenv("GLUTTON_TEST_POSTGRES_DSN")
	if len(dsn) == 0 {
		t.Skip("set GLUTTON_TEST_POSTGRES_DSN to run the postgres integration test")
	}
	ds := new(DatabaseSaver)
	assert.NoError(t, ds.Configure(&iface.Settings{SQLDriver: "postgres", SQLConnectionString: dsn}))
	defer ds.Close()

	payload := fmt.Sprintf("integration-%d", time.Now().UnixNano())
	assert.NoError(t, ds.Save(&iface.PayloadRecord{
		Payload:   payload,
		Timestamp: time.Now(),
		Remote:    "127.0.0.1",
		Meta:      map[string][]string{"Content-Type": {"application/json"}},
	}))

	db, err := sql.Open("postgres", dsn)
	assert.NoError(t, err)
	defer db.Close()

	var count int
	assert.NoError(t, db.QueryRow("SELECT count(*) FROM payload WHERE payload = $1", payload).Scan(&count))
	assert.Equal(t, 1, count)
}
