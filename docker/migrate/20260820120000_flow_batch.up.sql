CREATE TABLE
    flow_batch
    (
        nas_ip CHARACTER VARYING NOT NULL,
        file_name CHARACTER VARYING NOT NULL,
        PRIMARY KEY (nas_ip, file_name)
    );
