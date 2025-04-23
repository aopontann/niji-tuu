SET statement_timeout = 0;

--bun:split

CREATE TABLE "keywords" (
    "name" varchar(100) NOT NULL,
    "role_id" varchar(19) NOT NULL,
    "channel_id" varchar(30),
    "include" VARCHAR[],
    "ignore" VARCHAR[],
    "created_at" TIMESTAMP(0) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP(0) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY ("name")
);