# Vector iteration logs

One file per set, `art/vector/log/<set>.md`, named like the spec in [../specs/](../specs/). The log is the memory of the render, look and critique loop in [../STYLE.md](../STYLE.md): what each iteration saw and what it changed, so the next agent does not repeat a failed idea. Renders themselves stay outside the repo, under the run directory (`art-work/vector/<set>/iter-<n>/`); the log names them by that relative path.

## Format

Newest iteration at the bottom. Plain prose and lists, no tables.

```markdown
# <set>: iteration log

## Status

- resource-wood: done (iteration 3)
- resource-clay: needs-human (6 iterations; the 12 px brick seams still blob)
- ...

## Iteration <n> (<date>)

Renders: art-work/vector/<set>/iter-<n>/ (light, dark, DPR 1 and 2, grey, three CVD, peeps).
Lint: clean | <the remaining problems>.

### <icon id>

- Seen: what is wrong, at which size and in which column or context.
- Changed: what you changed in the SVG, and why.

## Blind check (<date>)

Reader: <who; "the author, not blind" if you drew or saw the labelled icons>.
Score: 12 px <k>/<n>, 16 px <k>/<n>; misses: <id read as id>, ...
```

Rules:

- Write the critique before you edit, from the PNGs you actually looked at.
- An icon gets at most 6 iterations. After the sixth, keep the best candidate and mark it `needs-human` in the Status list, with what still fails.
- Report the blind score honestly, with who read it.
