-- users table stores user balances
DROP TABLE IF EXISTS users;
CREATE TABLE users
(
    id      INT PRIMARY KEY,
    balance INT NOT NULL
);

-- reserves table store reserves for services
DROP TABLE IF EXISTS reserves;
CREATE TABLE reserves
(
    order_id   INT,
    user_id    INT REFERENCES users (id),
    service_id INT,
    cost       INT       NOT NULL,
    status     TEXT      NOT NULL,
    created    TIMESTAMP NOT NULL DEFAULT now(),
    PRIMARY KEY (order_id, user_id, service_id)
);

-- journal table stores all transactions
DROP TABLE IF EXISTS journal;
CREATE TABLE journal
(
    id      SERIAL PRIMARY KEY,
    user_id INT       NOT NULL REFERENCES users (id),
    amount  INT       NOT NULL,
    message TEXT      NOT NULL,
    created TIMESTAMP NOT NULL DEFAULT now()
);
