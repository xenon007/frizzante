# Get started

Create a new project using the starter template.

```bash
git clone https://github.com/razshare/frizzante-starter
```

> [!NOTE]
> Make sure you have [Go](https://go.dev/doc/install) and [Bun](https://bun.sh) installed.\
> If you'd rather use a different runtime than Bun to update your javascript dependencies,
> see [makefile, section "update"](https://github.com/razshare/frizzante-starter/blob/master/makefile#L1-L6).

Update dependencies

```bash
make update
```

Then start the server

```bash
make start
```

or build it

```bash
make build
```

> [!NOTE]
> The `www/dist` directory is embedded, which makes the final executable completely portable.
> 
> That being said, you can still create a "www/dist" directory near your executable.\
> Whenever a request is trying to access a file missing from the embedded file system, the server will fall 
> back to the nearby "www/dist" directory instead.
