# Visuals

Visual assets are generated with [Charm VHS](https://github.com/charmbracelet/vhs).

## Files

- Tapes live in `demo/`.
- `demo/golden.tape` creates the README-quality golden demo.
- `demo/navigation.tape`, `demo/success.tape`, and `demo/error.tape` create supporting GIFs.
- `demo/screenshots/*.tape` creates static PNG states.
- Generated assets live in `docs/assets/`.

Key outputs:

```text
docs/assets/golden-demo.gif
docs/assets/gifs/navigation.gif
docs/assets/gifs/success.gif
docs/assets/gifs/error.gif
docs/assets/screenshots/*.png
```

## Commands

Install or verify visual tooling, regenerate screenshots and GIFs, regenerate the golden demo, and validate expected outputs:

```bash
make plate
```

Focused helper targets:

```bash
make install-visual-tools
make visual-screenshots
make visual-gifs
make visual-golden
make visual-test
```

## Review Expectations

When UI, UX, layout, workflow, navigation, or styling changes:

1. Update or add VHS tapes under `demo/` when states or expected output change.
2. Use `--demo` in every tape.
3. Run `make plate`.
4. Review `docs/assets/golden-demo.gif`, `docs/assets/gifs/*.gif`, and `docs/assets/screenshots/*.png`.
5. Commit regenerated visual files when they changed.
