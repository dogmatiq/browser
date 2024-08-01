CREATE SCHEMA IF NOT EXISTS dogmabrowser;

CREATE TABLE IF NOT EXISTS dogmabrowser.repository (
    id             INT PRIMARY KEY,
    full_name      TEXT NOT NULL,
    commit_hash    TEXT NOT NULL,
    is_stale       BOOLEAN NOT NULL DEFAULT FALSE
);

CREATE INDEX IF NOT EXISTS repository_stale_idx
ON dogmabrowser.repository (is_stale);

CREATE TABLE IF NOT EXISTS dogmabrowser.type (
    id            SERIAL PRIMARY KEY,
    package       TEXT NOT NULL,
    name          TEXT NOT NULL,
    needs_removal BOOLEAN NOT NULL DEFAULT FALSE,
    repository_id INT,
    url           TEXT,
    docs          TEXT,

    UNIQUE (package, name),

    CONSTRAINT repository_fkey
        FOREIGN KEY (repository_id)
        REFERENCES dogmabrowser.repository (id)
        ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS type_repository_idx
ON dogmabrowser.type (repository_id);

CREATE TABLE IF NOT EXISTS dogmabrowser.application (
    key           TEXT PRIMARY KEY,
    name          TEXT NOT NULL,
    type_id       INT NOT NULL,
    is_pointer    BOOLEAN NOT NULL,
    repository_id INT NOT NULL,
    needs_removal BOOLEAN NOT NULL DEFAULT FALSE,

    CONSTRAINT repository_fkey
        FOREIGN KEY (repository_id)
        REFERENCES dogmabrowser.repository (id)
        ON DELETE CASCADE,

    CONSTRAINT type_fkey
        FOREIGN KEY (type_id)
        REFERENCES dogmabrowser.type (id)
        ON DELETE RESTRICT
);

CREATE INDEX IF NOT EXISTS application_repository_idx
ON dogmabrowser.application (repository_id);

CREATE INDEX IF NOT EXISTS application_type_idx
ON dogmabrowser.application (type_id);

CREATE TABLE IF NOT EXISTS dogmabrowser.handler (
    key             TEXT PRIMARY KEY,
    name            TEXT NOT NULL,
    application_key TEXT NOT NULL,
    handler_type    TEXT NOT NULL,
    type_id         INT NOT NULL,
    is_pointer      BOOLEAN NOT NULL,
    needs_removal   BOOLEAN NOT NULL DEFAULT FALSE,

    CONSTRAINT application_fkey
        FOREIGN KEY (application_key)
        REFERENCES dogmabrowser.application (key)
        ON DELETE CASCADE,

    CONSTRAINT type_fkey
        FOREIGN KEY (type_id)
        REFERENCES dogmabrowser.type (id)
        ON DELETE RESTRICT
);

CREATE INDEX IF NOT EXISTS handler_application_idx
ON dogmabrowser.handler (application_key);

CREATE INDEX IF NOT EXISTS handler_type_idx
ON dogmabrowser.handler (type_id);

CREATE TABLE IF NOT EXISTS dogmabrowser.handler_message (
    handler_key   TEXT NOT NULL,
    type_id       INT NOT NULL,
    is_pointer    BOOLEAN NOT NULL,
    role          TEXT NOT NULL,
    is_produced   BOOLEAN NOT NULL DEFAULT FALSE,
    is_consumed   BOOLEAN NOT NULL DEFAULT FALSE,
    needs_removal BOOLEAN NOT NULL DEFAULT FALSE,

    PRIMARY KEY (handler_key, type_id, is_pointer),

    CONSTRAINT handler_fkey
        FOREIGN KEY (handler_key)
        REFERENCES dogmabrowser.handler (key)
        ON DELETE CASCADE,

    CONSTRAINT type_fkey
        FOREIGN KEY (type_id)
        REFERENCES dogmabrowser.type (id)
        ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS handler_message_type_idx
ON dogmabrowser.handler_message (type_id);

CREATE INDEX IF NOT EXISTS handler_message_produced_idx
ON dogmabrowser.handler_message (is_produced, type_id);

CREATE INDEX IF NOT EXISTS handler_message_consumed_idx
ON dogmabrowser.handler_message (is_consumed, type_id);
