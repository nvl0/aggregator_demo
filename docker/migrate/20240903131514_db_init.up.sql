CREATE TABLE
    online_session
    (
        ip CHARACTER VARYING NOT NULL,
        sess_id INTEGER NOT NULL,
        nas_ip CHARACTER VARYING NOT NULL,
        CONSTRAINT sess_id_ix1 UNIQUE (sess_id),
        PRIMARY KEY (ip)
    );

INSERT INTO online_session (ip, sess_id, nas_ip) VALUES ('127.0.0.1', 1, '127.0.0.0');
INSERT INTO online_session (ip, sess_id, nas_ip) VALUES ('192.168.0.1', 2, '192.168.0.0');

CREATE TABLE
    channel
    (
        channel_id SERIAL NOT NULL,
        enabled BOOLEAN DEFAULT false NOT NULL,
        descr CHARACTER VARYING NOT NULL,
        CONSTRAINT descr_ix1 UNIQUE (descr),
        PRIMARY KEY (channel_id)
    );

INSERT INTO channel (descr) VALUES ('internal');
INSERT INTO channel (enabled, descr) VALUES (true, 'external');

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


