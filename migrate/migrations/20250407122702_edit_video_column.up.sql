SET statement_timeout = 0;

--bun:split
ALTER TABLE videos DROP COLUMN viewers;

--bun:split
ALTER TABLE videos DROP COLUMN announced;

--bun:split
ALTER TABLE videos DROP COLUMN thumbnail;

