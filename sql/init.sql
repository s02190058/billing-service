DROP TABLE IF EXISTS users;
CREATE TABLE users
(
    id      INT PRIMARY KEY,
    balance INT NOT NULL CHECK (balance >= 0)
);


DROP TABLE IF EXISTS journal;
CREATE TABLE journal
(
    id      SERIAL PRIMARY KEY,
    user_id INT       NOT NULL REFERENCES users (id),
    amount  INT       NOT NULL,
    message TEXT,
    created TIMESTAMP NOT NULL DEFAULT now()
);

DROP TABLE IF EXISTS orders;
CREATE TABLE orders
(
    id         INT,
    service_id INT,
    user_id    INT,
    amount     INT       NOT NULL CHECK (amount > 0),
    status     TEXT      NOT NULL,
    created    TIMESTAMP NOT NULL DEFAULT now(),
    PRIMARY KEY (id, service_id, user_id)
);
