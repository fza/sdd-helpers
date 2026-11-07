# Documentation

Every page states what the tree does, in the present tense.

Each piece has a folder with two pages: `usage.md`, what an agent or a person does with it, and
`architecture.md`, how it works inside.

| Folder | Piece |
|---|---|
| [`plugin/`](plugin/usage.md) | the plugin itself: how a project opts in, the skills, the options, what each session start says, developing on a checkout; the launcher, its fetch and lock, the release ([architecture](plugin/architecture.md)) |
| [`grounding-gate/`](grounding-gate/usage.md) | `sdd-grounding-gate`: the reads a turn owes before a question, a draft or a capture, the credential scan on every graph write, and the collapse of sdd output an agent was already shown ([architecture](grounding-gate/architecture.md)) |
| [`capture/`](capture/usage.md) | `sdd-capture`: writing one sdd entry from a draft ([architecture](capture/architecture.md)) |
