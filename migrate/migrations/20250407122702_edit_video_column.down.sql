SET statement_timeout = 0;

--bun:split
ALTER TABLE videos ADD COLUMN viewers integer NOT NULL;

--bun:split
ALTER TABLE videos ADD COLUMN announced boolean DEFAULT false;

--bun:split
ALTER TABLE videos ADD COLUMN thumbnail varchar NOT NULL DEFAULT '';

