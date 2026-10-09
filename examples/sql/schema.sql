-- Schema required by the default DatabaseSaver layout:
--   INSERT INTO payload(ts, remote, meta, payload) VALUES ($1, $2, $3, $4)
-- `meta` holds the request metadata serialized as json.
CREATE TABLE IF NOT EXISTS payload (
    ts      timestamptz NOT NULL,
    remote  text        NOT NULL,
    meta    jsonb       NOT NULL,
    payload text        NOT NULL
);
