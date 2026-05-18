# VHS recording scripts

These [`.tape`](https://github.com/charmbracelet/vhs) files generate the screenshots and GIFs used in the project README.

## Install VHS

```bash
brew install vhs
# or
go install github.com/charmbracelet/vhs@latest
```

## Generate

```bash
# from repo root
vhs docs/vhs/list.tape    # connection list demo
vhs docs/vhs/sftp.tape    # SFTP browser demo
vhs docs/vhs/form.tape    # add connection form

# all at once
for tape in docs/vhs/*.tape; do vhs "$tape"; done
```

Outputs land next to the tapes as `.gif` files.

## Adjusting

Each tape sets `Set FontSize`, `Set Width`, `Set Height`, `Set Theme`, and timing. Tweak them at the top of the file. See [VHS docs](https://github.com/charmbracelet/vhs#settings) for full options.

For PNG snapshots instead of animated GIFs, swap `Output foo.gif` for `Output foo.png` and remove most of the timed key presses (keep only the final state you want captured).

## Tips for the SFTP tape

`sftp.tape` actually connects to a remote — it'll need at least one valid connection in your local tamagosh config. Either:
- Run it on a machine that already has connections configured, or
- Add a test connection (e.g., a local Docker SSH container) before recording.
