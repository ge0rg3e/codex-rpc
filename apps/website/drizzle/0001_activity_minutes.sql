CREATE TABLE IF NOT EXISTS "activity_minutes" (
	"user_id" text NOT NULL,
	"minute" bigint NOT NULL,
	CONSTRAINT "activity_minutes_user_id_minute_pk" PRIMARY KEY("user_id","minute"),
	CONSTRAINT "activity_minutes_user_id_users_id_fk" FOREIGN KEY ("user_id") REFERENCES "public"."users"("id")
);
