# folderMaker

Creates a dated folder structure by asking a few questions. The questions and folder layout are fully defined in a JSON config file.

## Install

**With Go:**
```bash
go install github.com/Puhi8/folderMaker/cmd/folderMaker@latest
```

**Download binary** (from [GitHub Releases](https://github.com/Puhi8/folderMaker/releases/latest)):
```bash
# Linux (amd64)
curl -L https://github.com/Puhi8/folderMaker/releases/latest/download/folderMaker-linux-amd64.tar.gz | tar -xz
mv folderMaker-linux-amd64 ~/.local/bin/folderMaker
```

**Build from source:**
```bash
go build -o folderMaker ./cmd/folderMaker/
cp folderMaker ~/.local/bin/
```

## Usage

```bash
folderMaker <config.json> [flags]
folderMaker compile <template-dir> [out.json]
folderMaker expand  <config.json>  <out-dir>
folderMaker help    [command]
```

### Running a config

```bash
folderMaker adventure.json
folderMaker adventure.json -date 2026_05_22 -name Hike -c 1 -g 0 -d 0
```

Any question that has a `flag` field can be pre-answered from the command line so the program skips the prompt. If the supplied value is invalid the tool falls back to asking interactively.

### compile — directory tree → JSON
```bash
folderMaker compile ./my-template out.json
```

Reads a template directory and generates a config JSON. A directory that contains a config file becomes a section — the user will be asked to choose among its subdirectories. A directory without a config file is always created.

Config file format:
```
prompt: Which type of items?
multi:  true
flag:   items
header: ======= Title =======
header: Second header line
```

### expand — JSON → directory tree

```bash
folderMaker expand adventure.json ./my-template
```

Inverse of `compile`. Recreates the template directory structure from a config JSON so it can be edited and compiled back. Sections are named after their `flag` (or `section_N` if unset).

> Note: `shared` refs are not inlined — edit those sections of the template manually after expanding.

---

## Config JSON reference

`template.json` in this repo is a fully annotated starting point covering every feature.

The file has four top-level keys: `shared`, `base_dir`, `create`, and `sections`.

---

### `shared` — reusable entries

Define a directory list or a question once and reference it anywhere with `{ "ref": "name" }`.

```json
"shared": {
  "commonDirs": ["Notes", "References", "Exports"],
  "envQuestion": {
    "header": ["What environment?"],
    "prompt": "Enter choice (0-2): ",
    "choices": ["City", "Wild", "Creative"],
    "flag": "ce",
    "on": [["Museum", "Parks"], ["Path", "River"], ["Macro", "Sunset"]]
  }
}
```

---

### `base_dir` — the root folder name

```json
"base_dir": {
  "questions": [ ... ],
  "format": "{date} ({name}{variant})"
}
```

`format` is a template — `{id}` is replaced with the answer to the question whose `id` (or `flag`) matches.

#### Question types (inferred from the presence of `choices`)

**Text question**

```json
{ "prompt": "Project name: ", "flag": "name" }
```

| Field | Required | Description |
|---|---|---|
| `prompt` | yes | Text shown to the user |
| `flag` | no | CLI flag that pre-answers this question |
| `id` | no | Key used in `format`. Defaults to `flag`, then `field0`, `field1`… |
| `validate` | no | `"date"` — must be exactly 10 characters |

**Choice question**

```json
{
  "id": "variant",
  "header": ["Is this a special variant?"],
  "prompt": "Enter choice (0 or 1): ",
  "choices": ["No", "Yes"],
  "values": ["", "-special"],
  "flag": "v"
}
```

| Field | Required | Description |
|---|---|---|
| `prompt` | yes | Text shown after the choice list |
| `choices` | yes | List of options (user types the index) |
| `header` | no | Lines printed above the choice list |
| `flag` | no | CLI flag that pre-answers this question |
| `id` | no | Key used in `format`. Defaults to `flag` |
| `values` | no | Value inserted into `format` for each choice. Defaults to the choice label |

---

### `create` — unconditional top-level folders

```json
"create": ["Assets", "Exports"]
```

These folders are always created inside the root folder, before any sections are asked.

---

### `sections` — folder-creating questions

Each section is a choice question. The selected index picks an entry from `on`.

```json
{
  "header": ["Which type?"],
  "prompt": "Enter choice (0, 1, or 2): ",
  "choices": ["None", "Type A", "Type B"],
  "multi": true,
  "flag": "t",
  "on": [null, "FolderA", ["FolderB-1", "FolderB-2"]]
}
```

| Field | Required | Description |
|---|---|---|
| `prompt` | yes | Text shown after the choice list |
| `choices` | yes | List of options |
| `on` | yes | One entry per choice (see below) |
| `header` | no | Lines printed above the choice list |
| `flag` | no | CLI flag that pre-answers this question |
| `multi` | no | `true` — allow selecting multiple indices at once |

#### `on` entry forms

| JSON value | What it does |
|---|---|
| `null` | Skip (do nothing) |
| `"FolderName"` | Create one folder |
| `["A", "B"]` | Create multiple folders |
| `{ ... }` | Full action object (see below) |

#### Full action object

```json
{
  "cd": "SubDir",
  "create": ["A", "B"],
  "steps": [ ... ]
}
```

| Field | Description |
|---|---|
| `create` | Folders to create in the current directory before cd-ing |
| `cd` | Create this folder and enter it — subsequent `steps` run inside it |
| `steps` | Ordered list of steps (see below) |

#### Steps

| Form | What it does |
|---|---|
| `["A", "B"]` | Create folders A and B in the current directory |
| `{ "cd": "Sub" }` | Create Sub and enter it for all following steps |
| `{ "create": ["A", "B"] }` | Create folders without changing directory |
| `{ "ref": "name" }` | Run the named entry from `shared` |
| `{ "prompt": ..., "choices": ..., "on": ... }` | Inline choice question |
