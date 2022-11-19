# Billing service

## Структура проекта

```
.
├── cmd
│   └── app
│       └── main.go
├── configs
│   └── main.yml
├── internal
│   ├── app
│   │   └── app.go
│   ├── config
│   │   └── config.go
│   ├── model
│   │   └── transaction.go
│   ├── service
│   │   ├── errors.go
│   │   ├── order.go
│   │   └── user.go
│   ├── storage
│   │   ├── order.go
│   │   └── user.go
│   └── transport
│       ├── errors.go
│       ├── middleware.go
│       ├── order.go
│       ├── response.go
│       ├── responseWriter.go
│       ├── router.go
│       └── user.go
├── pkg
│   ├── httpserver
│   │   └── server.go
│   ├── postgres
│   │   └── postgres.go
│   └── zaplogger
│       └── logger.go
└── sql
    └── init.sql
```

## Запуск

Перед самим запуском необходимо задать некоторые переменные окружения в файле
`.env` (см. `example.env`) и разместить этот файл в корне.

Далее, находясь в корне проекта, введите команду `make compose-up`,
которая запустит сервис. Для его остановки введите `make compose-down`.

## API Endpoints

1) `GET /users/{user_id}` - получить баланс пользователя; возвращает баланс
пользователя в теле ответа
2) `POST /users/{user_id}` - пополнить баланс пользователя; если пользователь
в таблице отсутствует, создаётся новая запись; принимает сумму пополнения в
теле запроса; возвращает изменённый баланс пользователя в теле ответа
3) `POST /users/{user_id}/transfer` - перевести определённую сумму другому
пользователю; принимает сумму перевода и идентификатор пользователя,
которому осуществляется перевод, в теле запроса; возвращает изменённый баланс
пользователя в теле ответа
4) `GET /users/{user_id}/transactions?order_field=amount&limit=2&offset=10` -
получить список транзакций пользователя; возвращает список транзакций в теле
ответа
5) `POST /orders/{order_id}/reserve` - зарезервировать деньги с баланса
пользователя для оплаты услуги; принимает идентификатор пользователя,
идентификатор услуги и её стоимость в теле запроса
6) `POST /orders/{order_id}/confirm` - подтвердить оплату услуги; принимает
идентификатор пользователя, идентификатор услуги и её стоимость в теле запроса
7) `POST /orders/{order_id}/reject` - отменить резервирование денег; принимает
идентификатор пользователя, идентификатор услуги и её стоимость в теле запроса
8) `GET /orders/report?year=2022&month=11` - создать отчёт по услугам за
определённый месяц; возвращает ссылку на отчёт в теле ответа

## Примеры использования

Пусть сервис запущен на порту `8081`.

Изначально все таблицы базы данных пустые:  

```shell
$ curl localhost:8081/users/1
# {"message":"user not found: 1"}
```

Пополним баланс пользователя 1:

```shell
$ curl -d '{"amount":1000}' localhost:8081/users/1
# {"balance":1000}
```

Проверим, что баланс записался:

```shell
$ curl localhost:8081/users/1
# {"balance":1000}
```

Попробуем сделать перевод:

```shell
$ curl -d '{"amount":100,"receiver_id":2}' localhost:8081/users/1/transfer
# {"message":"user not found: 2"}
```

Единственный способ добавить запись о балансе пользователя в таблицу - это
пополнить его:

```shell
$ curl -d '{"amount":500}' localhost:8081/users/2
# {"balance":500}
```

Повторим попытку:

```shell
$ curl -d '{"amount":100,"receiver_id":2}' localhost:8081/users/1/transfer
# {"balance":900}
```

А что если перевести больше денег, чем лежит на балансе:

```shell
$ curl -d '{"amount":1000,"receiver_id":2}' localhost:8081/users/1/transfer
# {"message":"insufficient funds: 900"}
```

Теперь поработаем с заказами. Зарезервируем деньги на счёте пользователя 1
для двух разных услуг одного заказа:

```shell
$ curl -d '{"user_id":1,"service_id":23,"cost":300}' localhost:8081/orders/387/reserve
# {"status":"reserved"}
$ curl -d '{"user_id":1,"service_id":14,"cost":100}' localhost:8081/orders/387/reserve
# {"status":"reserved"}
```

Посмотрим на баланс пользователя:

```shell
$ curl localhost:8081/users/1
# {"balance":500}
```

Подтвердим оплату первой услуги:

```shell
$ curl -d '{"user_id":1,"service_id":14,"cost":100}' localhost:8081/orders/387/confirm
# {"status":"confirmed"}
```

Это действие занесёт в журнал нужную информацию.

Отменим оплату другой услуги:

```shell
$ curl -d '{"user_id":1,"service_id":14,"cost":100}' localhost:8081/orders/387/reject
# {"status":"rejected"}
```

Деньги вернутся на счёт пользователя:

```shell
$ curl localhost:8081/users/1
# {"balance":600}
```

Выведем список транзакций пользователя 1, отсортированных по дате:

```shell
$ curl localhost:8081/users/1/transactions\?order_field=created
# [{"id":1,"user_id":1,"amount":1000,"message":"account replenishment","created":"2022-11-19T00:40:38.958854Z"},{"id":3,"user_id":1,"amount":-100,"message":"transfer to the user 2","created":"2022-11-19T00:43:36.301486Z"},{"id":5,"user_id":1,"amount":-300,"message":"payment for the service 23","created":"2022-11-19T00:49:34.334401Z"}]
```

Оплатим ещё несколько услуг:

```shell
$ curl -d '{"user_id":2,"service_id":23,"cost":50}' localhost:8081/orders/387/reserve
# {"status":"reserved"}
$ curl -d '{"user_id":2,"service_id":14,"cost":150}' localhost:8081/orders/387/reserve
# {"status":"reserved"}
$ curl -d '{"user_id":2,"service_id":23,"cost":50}' localhost:8081/orders/387/confirm
# {"status":"confirmed"}
$ curl -d '{"user_id":2,"service_id":14,"cost":150}' localhost:8081/orders/387/confirm
# {"status":"confirmed"}
```

Сгенерируем отчёт по всем оплаченным услугам за ноябрь 2022:

```shell
$ curl localhost:8081/orders/report\?year=2022\&month=11
# {"url":"localhost:8081/reports/2022-11.csv"}
```

Перейдя по этой ссылку, получим сам отчёт:

```shell
$ curl localhost:8081/reports/2022-11.csv
# service;total revenue
# 14;150
# 23;350
```