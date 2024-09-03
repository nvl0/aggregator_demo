CREATE TABLE
    session
    (
        ip CHARACTER VARYING NOT NULL,
        sess_id INTEGER NOT NULL,
        nas_ip CHARACTER VARYING NOT NULL,
        PRIMARY KEY (ip)
    );

INSERT INTO session (ip, sess_id, nas_ip) VALUES ('127.0.0.1', 2, '127.0.0.0');

CREATE TABLE
    channel
    (
        channel_id INTEGER NOT NULL,
        descr CHARACTER VARYING NOT NULL,
        PRIMARY KEY (channel_id)
    );

INSERT INTO channel (channel_id, descr) VALUES (0, 'internal');
INSERT INTO channel (channel_id, descr) VALUES (1, 'external');

CREATE TABLE
    chunk
    (
        chunk_id SERIAL NOT NULL,
        sess_id INTEGER NOT NULL,
        channel_id INTEGER NOT NULL,
        upload INTEGER NOT NULL,
        download INTEGER NOT NULL,
        CONSTRAINT chunk_pkey PRIMARY KEY (chunk_id),
        CONSTRAINT chunk_fk1 FOREIGN KEY (channel_id) REFERENCES "channel" ("channel_id")
    );


