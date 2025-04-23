SET statement_timeout = 0;

--bun:split

CREATE TABLE "roles" (
    "name" varchar(100) NOT NULL,
    "id" varchar(19) NOT NULL,
    "channel_id" varchar(30),
    "keywords" VARCHAR[],
    "exclusion_keywords" VARCHAR[],
    "created_at" TIMESTAMP(0) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP(0) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY ("name")
);
