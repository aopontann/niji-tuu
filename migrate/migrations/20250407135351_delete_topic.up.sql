SET statement_timeout = 0;

--bun:split
DROP SEQUENCE IF NOT EXISTS topics_id_seq;

--bun:split
DROP TABLE "user_topics";

--bun:split
DROP SEQUENCE IF NOT EXISTS user_topics_topic_id_seq;

--bun:split
DROP TABLE "topics";
