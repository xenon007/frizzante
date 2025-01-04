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
> The `ui` directory is not embedded into the final executable.
