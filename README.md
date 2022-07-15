# go-playground

Toying around with Go for my own reference.

- [streamproc](./streamproc) stream processing exercise and kubernetes local dev setup.
- [gochat](./gochat) chat application using WebSockets, gRPC and a terminal UI.
- [packex](./packex) go package explorer for integration with third-party tools like neovim.
- [typednil](./typednil) understanding the infamous typed nils.
- [typeswitch](./typeswitch) playing with typeswitches.
- [websockets](./websockets) hello-worldish WebSockets.
- [yamlparse](./yamlparse) parsing yaml.

Ideas:

- Implement same app with [sqlc](https://sqlc.dev), [ent](https://entgo.io) and [sqlx](https://github.com/jmoiron/sqlx). Pick migration lib for each case (init and 1+ migration). Document some ideas, pros and cons.
- Implement dev server with which you can do: `dev-server -p 8080 --static /=./public --proxy /api=http://backend:9000 --proxy /api2=http://backend2:9000`
