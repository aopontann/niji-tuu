SET statement_timeout = 0;

--bun:split
CREATE SEQUENCE IF NOT EXISTS topics_id_seq;

--bun:split
CREATE TABLE "public"."topics" (
    "id" int4 NOT NULL DEFAULT nextval('topics_id_seq'::regclass),
    "name" varchar(100) NOT NULL,
    "created_at" timestamp(0) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" timestamp(0) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY ("id")
);


--bun:split
CREATE SEQUENCE IF NOT EXISTS user_topics_topic_id_seq;

--bun:split
CREATE TABLE "public"."user_topics" (
    "user_token" varchar(1000) NOT NULL,
    "topic_id" int4 NOT NULL DEFAULT nextval('user_topics_topic_id_seq'::regclass),
    "created_at" timestamp(0) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" timestamp(0) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT "user_topics_user_token_fkey" FOREIGN KEY ("user_token") REFERENCES "public"."users"("token"),
    CONSTRAINT "user_topics_topic_id_fkey" FOREIGN KEY ("topic_id") REFERENCES "public"."topics"("id"),
    PRIMARY KEY ("user_token","topic_id")
);
