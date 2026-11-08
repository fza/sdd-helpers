# `.sdd/grounding.yaml` keys the reads follow

No file, no key → every default. The gate's note or refusal always prints the form owed.

- **`listing_prefix: area-`** → the listing owed is one topic carrying the prefix, read whole:
  `sdd view --layout "topic(<prefix><name>):as-list"`. Unset → every topic: `active:as-counts`.
- **`read_command`** → suggested reads use it in place of `sdd`; bare `sdd` still counts.
- **`read_suffix`** → printed after every suggested read, never required:
  `python3 scripts/graph.py search --term x --entry <entry>`.
- **`gather_command`** → one call runs all three: `--query` = semantic, `--term` = literal, each
  `--area <prefix><name>` = that listing, no `--area` = the `--entry` subject's areas. A flag left
  out → that mode still owed.
- **`show_depth`** → the depth every `show` owes:

  ```yaml
  show_depth:
    down: 3   # left out → 2; 0 → nothing owed that way
    up: 1     # left out → 1
  ```

- **`enabled: false`** → gate silent for every session in the project.
- **`cli_capture: true`** → `sdd-capture` allowed beside sdd's engine. Owner's setting.
- **`draft_lint`** → the body's lint: `sdd-helpers:capture`.

A listing is enormous, and another capture mid-pass invalidates it: read the first run whole, then
only what changed.
