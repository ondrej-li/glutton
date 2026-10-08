package saver

import (
	"database/sql"
	"encoding/json"

	"github.com/pkg/errors"

	"github.com/defectus/glutton/pkg/iface"
)

const defaultLayout = "INSERT INTO payload(ts, remote, meta, payload) VALUES ($1, $2, $3, $4)"

// DatabaseSaver can save payload to database.
type DatabaseSaver struct {
	db     *sql.DB
	layout string
}

// Configure configures this instance of DatabaseSaver. The connection is verified so that an unreachable database fails fast.
func (ds *DatabaseSaver) Configure(settings *iface.Settings) (err error) {
	ds.db, err = sql.Open(settings.SQLDriver, settings.SQLConnectionString)
	if err != nil {
		return errors.Wrap(err, "error opening database")
	}
	if err = ds.db.Ping(); err != nil {
		return errors.Wrap(err, "error connecting to database")
	}
	ds.layout = settings.SQLayout
	if len(ds.layout) == 0 {
		ds.layout = defaultLayout
	}
	return nil
}

// Save stored data into the database. The meta information is serialized to json because a map is not a valid SQL argument.
func (ds *DatabaseSaver) Save(payload *iface.PayloadRecord) error {
	meta, err := json.Marshal(payload.Meta)
	if err != nil {
		return errors.Wrap(err, "error serializing payload meta")
	}
	_, err = ds.db.Exec(ds.layout, payload.Timestamp, payload.Remote, string(meta), payload.Payload)
	return errors.Wrap(err, "error saving payload")
}

// Close releases the database connection.
func (ds *DatabaseSaver) Close() error {
	if ds.db == nil {
		return nil
	}
	return ds.db.Close()
}
