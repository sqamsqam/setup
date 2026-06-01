# Visual Demos

VHS tapes live in this directory and must use `--demo` so recordings are deterministic, safe, and free of dry-run labels.

- `golden.tape` is the canonical README-quality happy path.
- `navigation.tape`, `success.tape`, and `error.tape` generate supporting GIFs.
- `screenshots/*.tape` generate static PNG states.

Run `make review-ui` after UI, UX, navigation, layout, styling, or workflow changes. Generated outputs are written to `docs/assets/`.
