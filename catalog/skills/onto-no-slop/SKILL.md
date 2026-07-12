---
name: onto-no-slop
description: Remove AI writing patterns from prose. Use when drafting, editing, or reviewing any onto prose artifact — proposals, designs, ADRs, guides, verification reports, commit messages — to eliminate predictable AI tells.
metadata:
  trigger: Writing prose, editing drafts, reviewing content for AI patterns
  author: Hardik Pandya (https://hvpandya.com)
  source: adapted from stop-slop, https://github.com/hardikpandya/stop-slop (MIT); vendored into the onto framework as onto-no-slop
---

# onto-no-slop

Eliminate predictable AI writing patterns from prose. This is the onto
framework's build of the **stop-slop** skill by Hardik Pandya, unchanged in its
rules and extended with a short section on applying them inside onto.

## Core Rules

1. **Cut filler phrases.** Remove throat-clearing openers, emphasis crutches, and all adverbs. See [references/phrases.md](references/phrases.md).

2. **Break formulaic structures.** Avoid binary contrasts, negative listings, dramatic fragmentation, rhetorical setups, false agency. See [references/structures.md](references/structures.md).

3. **Use active voice.** Every sentence needs a human subject doing something. No passive constructions. No inanimate objects performing human actions ("the complaint becomes a fix").

4. **Be specific.** No vague declaratives ("The reasons are structural"). Name the specific thing. No lazy extremes ("every," "always," "never") doing vague work.

5. **Put the reader in the room.** No narrator-from-a-distance voice. "You" beats "People." Specifics beat abstractions.

6. **Vary rhythm.** Mix sentence lengths. Two items beat three. End paragraphs differently. No em dashes.

7. **Trust readers.** State facts directly. Skip softening, justification, hand-holding.

8. **Cut quotables.** If it sounds like a pull-quote, rewrite it.

## Quick Checks

Before delivering prose:

- Any adverbs? Kill them.
- Any passive voice? Find the actor, make them the subject.
- Inanimate thing doing a human verb ("the decision emerges")? Name the person.
- Sentence starts with a Wh- word? Restructure it.
- Any "here's what/this/that" throat-clearing? Cut to the point.
- Any "not X, it's Y" contrasts? State Y directly.
- Three consecutive sentences match length? Break one.
- Paragraph ends with punchy one-liner? Vary it.
- Em-dash anywhere? Remove it.
- Vague declarative ("The implications are significant")? Name the specific implication.
- Narrator-from-a-distance ("Nobody designed this")? Put the reader in the scene.
- Meta-joiners ("The rest of this essay...")? Delete. Let the essay move.

## Scoring

Rate 1-10 on each dimension:

| Dimension | Question |
|-----------|----------|
| Directness | Statements or announcements? |
| Rhythm | Varied or metronomic? |
| Trust | Respects reader intelligence? |
| Authenticity | Sounds human? |
| Density | Anything cuttable? |

Below 35/50: revise.

## Examples

See [references/examples.md](references/examples.md) for before/after transformations.

## Using this inside onto

onto phases write prose the reader keeps: `proposal.md`, `design.md`,
`notes.md`, ADR drafts, `verification.md`, guide updates, and commit messages.
Run these rules over every such artifact before its phase gate, so the record a
human reads later sounds like a human wrote it.

Two adjustments for technical artifacts, where precision outranks style:

- **Keep the term that is exact.** A spec that says a value MUST be rejected
  needs "MUST," not a softer synonym. Cut empty adverbs ("really," "simply"),
  not load-bearing ones ("atomically," "idempotently").
- **Keep a real contrast when it carries the meaning.** "Files win downward;
  gates win upward" states two facts. Drop the *manufactured* reversal ("not X,
  it's Y"), not a genuine distinction the reader needs.

Everything else applies as written: active voice, name the actor, be specific,
cut the throat-clearing, vary the rhythm, no em dashes.

## License

MIT. Original skill by Hardik Pandya (https://github.com/hardikpandya/stop-slop).
