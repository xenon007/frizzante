# Get started

Create a new project using the starter template.

```bash
git clone https://github.com/razshare/frizzante-starter
```

Configure the project
```bash
make configure
```

> [!NOTE]
> This will install [Bun](https://bun.sh).\
> If you'd rather use a different runtime see [makefile](https://github.com/razshare/frizzante-starter/blob/master/makefile), section `configure`.

Load dependencies

```bash
make load
```

And finally, either start or build your project

```bash
make start
```

```bash
make build
```

> [!NOTE]
> The `www/dist` directory is embedded, which makes the final executable completely standalone.\
> 
> That being said, you can still create a "www/dist" directory near your executable.\
> Whenever a request is trying to access a file missing from the embedded file system, the server will fall 
> back to the nearby "www/dist" directory instead.
