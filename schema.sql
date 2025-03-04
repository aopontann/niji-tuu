CREATE TABLE "vtubers" (
    "id" varchar(24) NOT NULL,
    "name" varchar NOT NULL,
    "item_count" integer DEFAULT 0,
    "created_at" TIMESTAMP(0) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP(0) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY ("id")
);

CREATE TABLE "videos" (
    "id" varchar(11) NOT NULL, 
    "title" varchar NOT NULL, 
    "duration" varchar NOT NULL, 
    "song" boolean DEFAULT false, 
    "viewers" integer NOT NULL, 
    "content" varchar NOT NULL, 
    "scheduled_start_time" timestamp, 
    "thumbnail" varchar NOT NULL, 
    "created_at" TIMESTAMP(0) NOT NULL DEFAULT CURRENT_TIMESTAMP, 
    "updated_at" TIMESTAMP(0) NOT NULL DEFAULT CURRENT_TIMESTAMP, 
    PRIMARY KEY ("id")
);

CREATE TABLE "users" (
    "token" varchar(1000) NOT NULL,
    "song" boolean DEFAULT false NOT NULL,
    "info" boolean DEFAULT false NOT NULL,
    "created_at" TIMESTAMP(0) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP(0) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY ("token")
);

CREATE TABLE "roles" (
	"name" varchar(100) NOT NULL,
	"id" varchar(19) NOT NULL,
    "webhook_url" varchar(150) NOT NULL,
	"created_at" TIMESTAMP(0) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP(0) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY ("name")
);
