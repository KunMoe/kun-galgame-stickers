-- Existing production table (kungalgame_sticker.sticker). IF NOT EXISTS so a
-- first migrate on the live database is a no-op besides recording version 1.
CREATE TABLE IF NOT EXISTS sticker (
    id SERIAL PRIMARY KEY,
    sid INTEGER NOT NULL DEFAULT 0,
    pid INTEGER NOT NULL DEFAULT 0,
    src TEXT NOT NULL DEFAULT '',
    game JSONB NOT NULL,
    loli JSONB NOT NULL,
    vndb INTEGER NOT NULL DEFAULT 0,
    describe TEXT NOT NULL DEFAULT '',
    created TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS sticker_sid_pid_key ON sticker (sid, pid);
