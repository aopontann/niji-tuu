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

CREATE TABLE "topics" (
	"id" SERIAL NOT NULL,
	"name" varchar(100) NOT NULL,
	"created_at" TIMESTAMP(0) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP(0) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY ("id")
);

CREATE TABLE "roles" (
	"name" varchar(100) NOT NULL,
	"id" varchar(19) NOT NULL,
    "webhook_url" varchar(150) NOT NULL,
	"created_at" TIMESTAMP(0) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP(0) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY ("name")
);

CREATE TABLE "video_topics" (
	"id" varchar(100) NOT NULL,
	"topic_id" varchar(100),
	"video_id" varchar(11),
	"created_at" TIMESTAMP(0) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP(0) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY ("id"),
    FOREIGN KEY ("topic_id") REFERENCES public.topics(id),
    FOREIGN KEY ("video_id") REFERENCES public.videos(id)
);

CREATE TABLE "user_topics" (
	"user_token" varchar(1000),
	"topic_id" SERIAL,
	"created_at" TIMESTAMP(0) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP(0) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY ("user_token", "topic_id"),
    FOREIGN KEY ("user_token") REFERENCES public.users(token),
    FOREIGN KEY ("topic_id") REFERENCES public.topics(id)
);

CREATE TABLE "request_topics" (
	"id" SERIAL NOT NULL,
	"name" varchar(100) NOT NULL,
    "user_token" varchar(1000),
	"created_at" TIMESTAMP(0) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP(0) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY ("id")
    FOREIGN KEY ("user_token") REFERENCES public.users(token),
);
