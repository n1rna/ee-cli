# Working with the `ee` environment variable manager

`ee` is a CLI that brings structure and validation to environment variables. It
uses a `.ee` project file (JSON), standard `.env` files, and optional schema
files to define, validate, hydrate, and deploy environment variables across
environments (development, staging, production, ...).

Use this guide whenever a task involves environment variables, `.env` files, a
`.ee` file, secrets, or configuration for a project that has (or should have)
`ee` set up.

## Is `ee` installed?

Check before running commands:

```bash
ee --version
```

If it is missing, install it:

```bash
curl -sSfL https://raw.githubusercontent.com/n1rna/ee-cli/main/install.sh | sh
# or
go install github.com/n1rna/ee-cli/cmd/ee@latest
```

## First: does this project already use `ee`?

Look for a `.ee` file in the project root.

- **`.ee` exists** → the project already uses `ee`. Jump to
  "Working with an existing `ee` project".
- **No `.ee` file** → jump to "Adding `ee` to a new project".

```bash
ls .ee 2>/dev/null && echo "ee is set up" || echo "no .ee file yet"
```

---

## Adding `ee` to a new project

Follow these steps when the project has no `.ee` file yet.

1. **Initialize the project.** From the project root:

   ```bash
   # Use the current directory name as the project name
   ee init

   # Or name it explicitly
   ee init my-api
   ```

   This creates a `.ee` file, plus sample `.env.development` and
   `.env.production` files, and a default inline schema (`NODE_ENV`, `PORT`,
   `DEBUG`) unless you provide your own.

2. **Define the schema.** Prefer a dedicated schema file for anything beyond a
   toy project. Create `schema.yaml`:

   ```yaml
   name: my-api
   description: API service configuration
   variables:
     - name: DATABASE_URL
       type: string
       title: Database connection string
       required: true
     - name: PORT
       type: number
       title: Server port
       required: false
       default: "3000"
     - name: API_KEY
       type: string
       title: API authentication key
       required: true
       regex: "^[a-zA-Z0-9_-]+$"
   ```

   Then reference it when initializing (or edit the `.ee` file's `schema`
   section to `{ "ref": "./schema.yaml" }`):

   ```bash
   ee init my-api --schema ./schema.yaml
   ```

   Alternatively, define variables inline without a file:

   ```bash
   ee init my-api \
     --var "DATABASE_URL:string:Database URL:true" \
     --var "PORT:number:Server port:false:3000"
   ```

   The `--var` format is `name:type:title:required[:default]`.

3. **Fill in the `.env` files.** Edit `.env.development` / `.env.production`
   (created by `ee init`) with real values. Keep secrets out of committed files
   (see "Handling secrets" below).

4. **Verify the setup.** Confirm the schema loads and required variables are
   present:

   ```bash
   ee verify --verbose
   # Auto-create missing files / append missing required vars:
   ee verify --fix
   ```

5. **Use it.** Run the app with an environment applied:

   ```bash
   ee apply development -- npm start
   ```

6. **Recommended `.gitignore` additions** (keep real secrets out of git):

   ```gitignore
   .env.local
   .env.*.local
   .env.secrets
   *.secret.env
   ```

   Commit `.ee`, `schema.yaml`, and non-secret `.env` files.

---

## Working with an existing `ee` project

When a `.ee` file is present, do **not** re-run `ee init`. Instead:

1. **Read the configuration.** Open the `.ee` file to learn the project name,
   schema location, defined environments, and any remote origins. Then confirm
   everything is consistent:

   ```bash
   ee verify --verbose
   ```

2. **List the environments** by inspecting the `environments` object in `.ee`
   (e.g. `development`, `staging`, `production`).

3. **Run commands with an environment applied** instead of manually exporting
   variables:

   ```bash
   # Start a subshell with the environment loaded
   ee apply development

   # Or run a single command with it
   ee apply production -- npm run build
   ```

4. **Inspect what an environment resolves to** without applying it (safe,
   read-only):

   ```bash
   ee apply production --dry-run
   ee apply production --dry-run --format json
   ee apply production --dry-run --format dotenv
   ```

5. **Add a new variable to the project:**
   - Add it to the schema (`schema.yaml` or the inline `schema.variables` in
     `.ee`).
   - Add its value to the relevant `.env` file(s).
   - Run `ee verify --fix` to append any missing required variables to the
     `.env` files with their defaults.

6. **Add a new environment:** add an entry under `environments` in `.ee`
   pointing at an `.env` file (single `env`) or multiple `sources`, then create
   the file(s) and run `ee verify`.

7. **Deploy / push secrets** to a configured remote origin (GitHub Actions
   secrets or Cloudflare Workers) — see "Pushing secrets to origins".

> Important for coding agents: never print or commit real secret values. When
> showing an environment, prefer `ee --mask` or `--dry-run` output and keep
> secret files gitignored.

---

## The `.ee` file format

The `.ee` file is JSON at the project root:

```json
{
  "project": "my-api",
  "schema": { "ref": "./schema.yaml" },
  "environments": {
    "development": { "env": ".env.development" },
    "production": {
      "sources": [".env", ".env.production", ".env.secrets"]
    }
  },
  "origins": {
    "github": {
      "type": "github",
      "mode": "bundled",
      "secret_name": "ENV_PRODUCTION",
      "repo": "myorg/my-app",
      "environment": "production"
    }
  }
}
```

- **`schema`** — either inline (`"variables": { ... }`) or a file reference
  (`"ref": "./schema.yaml"`). Refs accept relative paths, absolute paths, or
  `file://` URIs; plain filenames need a `.yaml`/`.yml`/`.json` extension.
- **`environments`** — each maps to a single `env` file or a `sources` array
  that is merged left-to-right (later values override earlier ones). A source
  can also be an inline `{ "KEY": "value" }` object.
- **`origins`** — remote push targets (`github`, `cloudflare`).

## Schema file format

```yaml
name: web-service
description: Schema for web service applications
variables:
  - name: DATABASE_URL
    title: Database connection URL
    type: string        # string | number | boolean | url
    required: true
  - name: PORT
    type: number
    required: false
    default: "3000"
  - name: API_KEY
    type: string
    required: true
    regex: "^[a-zA-Z0-9_-]+$"
```

Variable properties: `name` (required), `type` (`string`/`number`/`boolean`/
`url`), `title` (optional), `required` (bool), `default` (optional string),
`regex` (optional validation pattern).

## `.env` file format

Standard `KEY=VALUE`. Optional annotation comments document each variable and
are written by `ee init` / `ee verify --fix`:

```bash
# schema: ./schema.yaml

# title: Database connection URL
# type: string
# required: true
DATABASE_URL=postgres://localhost:5432/myapp

# title: Server port
# type: number
# default: 3000
PORT=3000
```

---

## Command reference

### `ee` (root) — inspect the current shell environment

```bash
ee                                  # print all env vars
ee --filter 'DB_*,DATABASE_*'       # wildcard include; separators , | /
ee --filter 'NODE*,!NODE_OPTIONS'   # prefix ! to exclude
ee --format json                    # env (default) | json | dotenv
ee --mask                           # mask sensitive values (KEY/SECRET/TOKEN/...)
```

### `ee init [project-name]` — create a new project

Flags: `-s/--schema <path>`, `--var <name:type:title:required[:default]>`
(repeatable), `-f/--force`, `-q/--quiet`.

### `ee apply <environment|file> [-- command [args...]]` — load an environment

Detects a file path when the argument starts with `.`, `/`, or `~`, or the file
exists; otherwise treats it as a project environment name (needs `.ee`). Without
a trailing command it starts a subshell. Flags: `-d/--dry-run`,
`-f/--format <env|dotenv|json>`, `-q/--quiet`. Alias: `ee a`.

### `ee verify` — validate the project

Checks the schema loads, every environment has an `.env` file, and required
variables are present. Flags: `--fix` (create missing files / append missing
required vars), `--verbose`, `--env <name>`, `--quiet`.

### `ee hydrate <environment>` — build an env file from the shell + schema

Resolves each schema variable from the current shell env, then the schema
default, then empty (warns for required). Flags: `-o/--output <path>`,
`-f/--format <dotenv|json|yaml>`. Useful in CI.

### `ee push [origin] <environment>` — push secrets to a remote origin

Pushes to GitHub Actions secrets or Cloudflare Workers. Flags: `--dry-run`,
`--mode <bundled|individual>`, `--quiet`. If only one origin is configured the
name can be omitted.

### `ee auth [tool]` — check origin CLI authentication

Checks `gh` (GitHub) and `wrangler` (Cloudflare). Run before `ee push`.

### `ee skill <agent>` — install this guide for a coding agent

Writes this usage guide into the convention expected by the selected coding
agent (`claude`, `cursor`, `copilot`, `codex`, `opencode`, or `all`).

---

## Handling secrets

- Keep secret values in a gitignored file such as `.env.secrets` and stack it
  via `sources`:

  ```json
  { "environments": { "production": {
    "sources": [".env", ".env.production", ".env.secrets"] } } }
  ```

- Never commit real secrets. Never echo secret values into logs, chat, or
  commits. Use `ee --mask` or `--dry-run` when you need to show a config.

## Pushing secrets to origins

```bash
ee auth gh                          # verify GitHub CLI is authenticated
ee push github production --dry-run # preview
ee push github production           # push (bundled multi-line secret)

ee auth wrangler                    # verify Cloudflare wrangler
ee push cloudflare production       # push individual secrets to a Worker
```

- **bundled** (GitHub default): all vars combined into one multi-line
  `KEY=VALUE` secret, consumable by the [`ee-action`](https://github.com/n1rna/ee-action)
  GitHub Action.
- **individual** (Cloudflare default): each variable pushed separately.

## CI / GitHub Actions

Hydrate an env file at deploy time from a bundled secret:

```yaml
- uses: actions/checkout@v4
- name: Prepare environment
  uses: n1rna/ee-action@v1
  with:
    environment: production
    config_path: .ee
    env_file: .env.production
    gh_secret: ${{ secrets.ENV_PRODUCTION }}
- run: docker build --env-file .env.production -t myapp .
```

## Quick recipes

```bash
# Compare two environments
diff <(ee apply development --dry-run) <(ee apply production --dry-run)

# Export an environment to a file
ee apply production --dry-run --format dotenv > .env.prod

# Audit secrets in the current shell (masked)
ee --filter '*KEY*,*SECRET*,*TOKEN*,*PASSWORD*' --mask

# Validate before deploying
ee verify --env production
```
