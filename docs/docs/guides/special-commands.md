---
title: Special Commands
description: Backslash commands and built-in commands available inside the pgxcli REPL.
keywords: [psql backslash commands, list tables postgres, describe table postgres, psql commands, session commands]
sidebar_position: 5
---

pgxcli supports PostgreSQL-style backslash commands. Type them directly at the prompt.

---

## Session Commands

These control your session:

| Command | Description |
|---------|-------------|
| `\q`, `\quit`, `\exit` | Quit pgxcli (case-insensitive) |
| `\c <database>` | Switch to a different database on the same server |
| `\connect <database>` | Same as `\c` |
| `\conninfo` | Show current connection details (database, user, host, port) |

### Switching Databases

```
\c other_db
```

pgxcli closes the current connection and opens a new one to `other_db`. The server, user, and port stay the same.

### Connection Info

```
\conninfo
```

Outputs something like:

```
You are connected to database "mydb" as user "postgres" on Host "localhost" at port 5432
```

---

## Catalog Commands

These come from the [pgxspecial](https://github.com/balajz/pgxspecial) library and work like their `psql` equivalents. pgxcli provides **autocompletion** for these meta commands to help you discover them quickly:

| Command | Description |
|---------|-------------|
| `\d [pattern]` | Describe a table, view, or other object |
| `\dt [pattern]` | List tables |
| `\dv [pattern]` | List views |
| `\di [pattern]` | List indexes |
| `\ds [pattern]` | List sequences |
| `\df [pattern]` | List functions |
| `\l` | List all databases |
| `\dn` | List schemas |
| `\du` | List roles |
| `\dx` | List installed extensions |

:::tip
Commands with `[pattern]` accept an optional filter. For example, `\dt public.*` lists only tables in the `public` schema.
:::

---

## Change the Working Directory

Use `\cd` to change the working directory used for local paths:

```text
\cd migrations
\cd ../sql
\cd "directory with spaces"
```

Running `\cd` without a directory changes to your home directory. Paths may be absolute, relative to the current working directory, or start with `~` to refer to your home directory.

Press `Tab` after `\cd` to complete directory names. Hidden directories are suggested when the name being completed starts with `.`.

---

## Edit SQL in an External Editor

Use `\e` or `\edit` to open SQL in an external editor:

```text
\e
\e query.sql
\edit "query notes.sql"
```

Without a filename, the editor starts with the most recently submitted SQL, or an empty file when no SQL has been submitted yet. With a filename, pgxcli opens that file instead. After you save and exit, the edited text is returned to the prompt for review; it is not executed automatically.

Relative filenames use the current working directory, including changes made with `\cd`. Press `Tab` after `\e` or `\edit` to complete files and directories.

pgxcli chooses the editor from `$PSQL_EDITOR`, `$EDITOR`, then `$VISUAL`. If none is set, it uses `vi` on Unix-like systems or `notepad.exe` on Windows.

---

## Execute SQL from a File

Use `\i` or `\include` to execute SQL statements from a file:

```text
\i migrations/create_tables.sql
\include "queries/monthly report.sql"
```

Statements execute immediately through the normal query runner and follow the configured `on_error` behavior. Relative filenames use the current working directory, including changes made with `\cd`; paths may also start with `~`.

Press `Tab` after `\i` or `\include` to complete files and directories.

:::note
Included files are currently treated as SQL only. Backslash commands, nested includes, psql variables, conditionals, and `\i -` are not supported inside included files.
:::

---

## Built-in Commands

These are pgxcli-specific:

| Command | Description |
|---------|-------------|
| `\clear`, `/clear` | ~~Clear the terminal screen~~ (**Currently disabled**, will be fixed in an upcoming update) |

---

## SQL Execution

Anything that isn't a special command is treated as SQL and sent to the database.
