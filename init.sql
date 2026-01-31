CREATE TABLE gateway_routes (
    id SERIAL PRIMARY KEY,
    path TEXT UNIQUE NOT NULL
);

CREATE TABLE gateway_backends (
    id SERIAL PRIMARY KEY,
    route_id INT REFERENCES routes(id) ON DELETE CASCADE,
    target_url TEXT NOT NULL,
    is_active BOOLEAN DEFAULT true
);

INSERT INTO gateway_routes(path) VALUES('/blog');

INSERT INTO gateway_backends(route_id,target_url)
VALUES
(1,'https://api-blog.labkita.my.id'),
(1,'https://oracle-api-blog.labkita.my.id'),
(1,'https://claw-api-blog.labkita.my.id');
