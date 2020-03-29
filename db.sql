create schema shard_1;
create sequence shard_1.global_id_sequence;

CREATE OR REPLACE FUNCTION shard_1.id_generator(OUT result bigint) AS $$
DECLARE
    our_epoch bigint := 1314220021721;
    seq_id bigint;
    now_millis bigint;
    -- the id of this DB shard, must be set for each
    -- schema shard you have - you could pass this as a parameter too
    shard_id int := 1;
BEGIN
    SELECT nextval('shard_1.global_id_sequence') % 1024 INTO seq_id;

    SELECT FLOOR(EXTRACT(EPOCH FROM clock_timestamp()) * 1000) INTO now_millis;
    result := (now_millis - our_epoch) << 23;
    result := result | (shard_id << 10);
    result := result | (seq_id);
END;
$$ LANGUAGE PLPGSQL;


CREATE TABLE message ( 
  chat_id bigint not null,
  message_id bigint not null DEFAULT shard_1.id_generator(),
  u_id bigint not null,
  text TEXT,
  created_at TIMESTAMP DEFAULT now()
);


CREATE TABLE ranking ( 
  ranking_id SERIAL PRIMARY KEY,
  chat_id bigint not null,
  message_id bigint not null,
  u_id bigint not null,
  lang char(2) not null,
  ranking int not null,
  created_at TIMESTAMP DEFAULT now()
);

CREATE TABLE botstate ( 
  chatkey char(32) PRIMARY KEY,
  chat_id bigint not null,
  u_id bigint not null,
  state TEXT,
  text TEXT
);

CREATE UNIQUE INDEX ranking_norepeat_idx ON ranking (chat_id, u_id, message_id, lang);

GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO localapp;
GRANT ALL PRIVILEGES ON SCHEMA shard_1 TO localapp;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA shard_1 TO localapp;


CREATE TABLE session ( 
  chatkey char(32) PRIMARY KEY,
  chat_id bigint not null,
  u_id bigint not null,
  data bytea
);


