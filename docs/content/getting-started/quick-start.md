---
title: "Quick start"
description: "Run your first s2 command."
weight: 30
---

Once `s2` is on your `PATH`:

```bash
s2 --help       # see the command tree
s2 version      # build info
```

This is a fresh scaffold, so the command tree is just `version` for now. Add
your first real command in `cli/`, build on the `semanticscholar` library package,
and document it here.

A good first command usually fetches one thing and prints it as JSON, so the
output pipes straight into `jq` and the rest of your tools.
