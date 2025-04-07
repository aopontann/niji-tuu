CREATE TABLE "vtubers" (
    "id" varchar(24) NOT NULL,
    "name" varchar NOT NULL,
    "item_count" integer DEFAULT 0,
    "playlist_latest_url" varchar,
    "created_at" TIMESTAMP(0) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP(0) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY ("id")
);

CREATE TABLE "videos" (
    "id" varchar(11) NOT NULL,
    "title" varchar NOT NULL,
    "duration" varchar NOT NULL,
    "viewers" integer NOT NULL,
    "content" varchar NOT NULL,
    "announced" boolean DEFAULT false,
    "scheduled_start_time" timestamp,
    "thumbnail" varchar NOT NULL DEFAULT '',
    "created_at" TIMESTAMP(0) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP(0) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY ("id")
);

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

CREATE TABLE "users" (
    "token" varchar(1000) NOT NULL,
    "song" boolean NOT NULL DEFAULT false,
    "info" boolean NOT NULL DEFAULT false,
    "created_at" TIMESTAMP(0) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP(0) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY ("token")
);
