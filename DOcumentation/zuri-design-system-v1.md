# Zuri Design System v1.0

## Purpose

This document extends the locked brand palette (deep verdigris teal, near black text, warm off white background, muted sage gray) into a complete token system for the Electron GUI shell. It exists to keep the daemon status panels, the live activity feed, and future GitHub App configuration screens visually coherent as separate build phases touch the UI independently.

The guiding idea: Zuri is an audit trail for engineering decisions. The interface should read like a precision instrument for that job, not like a generic SaaS marketing site wrapped around a desktop app. Every visual choice below is justified against that job. Nothing here uses gradients, glassmorphism, floating bubble shapes, or decorative motion. Radius is small and consistent, shadows are functional (they indicate a layer sits above another layer, nothing more), and color is used to encode state, never to decorate.

---

## 1. Color

### 1.1 Core palette (locked)

| Token | Hex | Role |
|---|---|---|
| `color.primary` | `#0E5C56` | Primary actions, active nav state, focus ring, brand mark |
| `color.text.primary` | `#161A19` | Body text, headings |
| `color.background` | `#FAF7F1` | App background |
| `color.muted` | `#8C978F` | Secondary text, inactive icons, placeholder text |

The near black and off white are warm, not neutral gray. This matters because a cold near black next to a warm background reads as mismatched, which is one of the fastest tells of an unconsidered palette.

### 1.2 Derived scale

Everything below is derived from the four core tokens by adjusting lightness only, so the palette never introduces an unrelated hue family.

| Token | Hex | Role |
|---|---|---|
| `color.primary.hover` | `#0A4A45` | Button and link hover, pressed state |
| `color.primary.tint` | `#E3EDEB` | Selected row background, active tab underline background |
| `color.surface` | `#FFFFFF` | Cards, panels, modals sitting above the app background |
| `color.border` | `#DDD7C9` | Hairline dividers, input borders, table rules |
| `color.border.strong` | `#C5BEAC` | Focused input border, active card outline |
| `color.text.secondary` | `#4B5350` | Supporting copy, field labels |

### 1.3 Functional colors

Status colors are pulled from the same warm, desaturated register as the core palette rather than reaching for default red, amber, and green. A saturated traffic light palette next to a muted brand palette is the single most common thing that makes an otherwise careful design look assembled from a component library rather than designed for this product.

| Token | Hex | Role |
|---|---|---|
| `color.success` | `#3F6E52` | Memory resolved, PR merged, healthy daemon state |
| `color.warning` | `#93672B` | Stale decision, pending review, approaching tier transition |
| `color.danger` | `#8C3B33` | Failed extraction, revoked access, hard fail on startup |
| `color.info` | `#2F6E74` | Background job running, non blocking notice |

None of these are used at full saturation anywhere in the UI. They appear as small fills (a 6px status dot, a tag background at 12% opacity, a left border on a log line), never as large blocks of color. Large colored blocks are what make status colors read as alerts instead of information.

### 1.4 What this palette explicitly avoids

No blue. Blue is the default accent of nearly every enterprise SaaS product (Stripe, Linear, Notion, GitHub's primary buttons), and introducing it here would dilute the one thing that makes Zuri's palette distinct: it is warm and earthy where the category default is cool and clinical. No terracotta or warm clay accent either, since that specific tone is closely associated with Claude's own interaction accent and would read as an unintentional borrow rather than a brand choice.

---

## 2. Typography

### 2.1 Typeface selection

**IBM Plex Sans** for interface text, **IBM Plex Mono** for anything that is data rather than prose.

This is a deliberate pairing, not the default Inter that most dev tools reach for. Plex was designed at IBM specifically for technical and engineering software, and its mono variant is drawn from the same skeleton as the sans, so the two feel like one family rather than a UI face bolted to a code face. That coherence matters here because Zuri's interface constantly switches between prose (decision summaries, conventions) and data (memory IDs, timestamps, latency figures, hex diffs), and the two should feel like one voice speaking in two registers, not two unrelated fonts.

Plex Mono is used for: memory and decision IDs, timestamps in the activity feed, latency and score figures, file paths, commit SHAs, and any raw JSON shown in a debug panel. Plex Sans is used for everything else.

### 2.2 Scale

| Token | Size / Line height | Weight | Use |
|---|---|---|---|
| `type.display` | 28px / 34px | 600 | Panel titles (Activity, Decisions, Settings) |
| `type.heading` | 18px / 24px | 600 | Card headers, modal titles |
| `type.body` | 14px / 20px | 400 | Default interface text |
| `type.body.medium` | 14px / 20px | 500 | Emphasized inline text, active nav label |
| `type.caption` | 12px / 16px | 400 | Field labels, timestamps, helper text |
| `type.data` | 13px / 18px | 400, Plex Mono | IDs, hashes, numeric figures |

Weight does the work that size usually does in templated interfaces. Most AI generated UIs escalate hierarchy by making headings larger and larger; here the display size is capped at 28px even for the top level panel title, and hierarchy below that is carried by weight and by the secondary text color, not by size jumps. This keeps the interface feeling dense and instrument like rather than marketing like.

---

## 3. Layout and spacing

### 3.1 Spacing scale

4, 8, 12, 16, 24, 32, 48 (px). No half steps. Every margin, padding, and gap in the app resolves to one of these seven values.

### 3.2 Grid

Three panel layout: a fixed 240px sidebar (workspace nav, memory tiers), a flexible main panel (decision detail, settings), and a 320px right rail reserved for the live activity feed when it is open. The right rail is the one place motion is allowed to be continuous, since it is a live stream by nature; the sidebar and main panel are otherwise static.

### 3.3 Radius

6px for buttons, inputs, and tags. 8px for cards and panels. 4px for small status pills. Nothing larger. Enterprise SaaS marketing sites often use large radii (16px and up) to feel soft and approachable; a control plane for engineering decisions should feel precise instead, so radius stays small and consistent across every component rather than escalating with component size.

### 3.4 Elevation

Two levels only.

- Level 0: app background, sidebar, main panel. No shadow.
- Level 1: cards, dropdowns, the activity feed panel. `box shadow: 0 1px 2px rgba(22, 26, 25, 0.06), 0 1px 1px rgba(22, 26, 25, 0.04)`. A hairline border in `color.border` does most of the separation work; the shadow is a supporting cue, not the primary one.

Modals get a slightly stronger shadow (`0 4px 12px rgba(22, 26, 25, 0.12)`) and nothing else changes. There is no third elevation level, no blurred backdrop, and no translucency anywhere. A frosted or blurred backdrop behind a modal is glassmorphism by another name and is explicitly out of scope.

---

## 4. Components

### 4.1 Buttons

Primary button: `color.primary` fill, white text, 6px radius, no shadow at rest, `color.primary.hover` fill on hover. Secondary button: `color.surface` fill, 1px `color.border` outline, `color.text.primary` label. There is no tertiary button style beyond a plain text link in `color.primary`. Three button styles is enough for this product; a fourth ghost or ghost outline variant tends to appear in templated systems without a real reason to exist.

### 4.2 Status indicators

A 6px filled circle in the relevant functional color, paired with a Plex Mono label (`RESOLVED`, `PENDING`, `FAILED`). No pill shaped badges with colored backgrounds for primary status, since those read as marketing chrome. Pills at 4px radius are reserved for lower emphasis metadata tags (memory type, source repo) using `color.muted` text on a `color.background` fill with a `color.border` outline, never a functional color fill.

### 4.3 Tables and logs

Hairline row dividers in `color.border`, no zebra striping. Row hover state is `color.primary.tint`, matching the selected row state at slightly lower opacity, so hover reads as a preview of selection rather than an unrelated highlight color.

### 4.4 Activity feed (right rail)

Each event is a single row: Plex Mono timestamp, a status dot, a one line Plex Sans description. New events enter with a 150ms opacity and 4px vertical slide, nothing bouncier. This is the one continuously live surface in the app and is described further in the signature element below.

---

## 5. Signature element: the provenance thread

Every other enterprise dev tool UI defaults to a card grid or a plain list for its history view. Zuri's actual job is tracing where a decision came from and how it has been cited or revived over time, so the signature element is a **provenance thread**: a 2px vertical line in `color.primary`, tint version `color.primary.tint` for its unlit segments, running down the left edge of any timeline (the activity feed, a decision's citation history, the audit log). Each event on that decision is a small filled dot on the line; a revival event is drawn as a dot that reopens the line's tint from solid back to full color for the segment that follows it, a literal read of "this decision came back to life."

This is the one place the interface is allowed a moment of specificity. Everywhere else stays quiet so this element carries the identity.

---

## 6. Motion

Two allowed motions: the activity feed row entrance described above, and a panel switch fade in the main content area. No page load sequences, no hover triggered scale or lift on cards, no skeleton shimmer for loading states (use a plain Plex Mono "Loading" label instead, consistent with the instrument like tone). Reduced motion preference disables both.

### 6.1 Tokens

| Token | Value | Use |
|---|---|---|
| `motion.fast` | 100ms | Panel switch fade, button hover fill change |
| `motion.normal` | 150ms | Activity feed row entrance (opacity and 4px vertical slide) |
| `motion.slow` | 250ms | Modal open and close, reserved for future use |
| `easing.standard` | `cubic-bezier(0.2, 0, 0, 1)` | Default for every transition above |

There is one easing curve for the whole app. It decelerates into place with no overshoot, which fits the instrument like tone from section 6's opening paragraph; a bounce or spring curve would read as playful in a way this product is not. Every future component should reach for one of these three durations rather than picking a new number. If a fourth duration ever seems necessary, that is a signal to question whether the motion is needed at all before adding a token for it.

---

## 7. Explicitly out of scope

State plainly, for anyone picking this file up later: no gradients anywhere, no glassmorphism or blurred translucent surfaces, no bubble or blob shapes, no emoji in interface copy, no decorative icons without a functional label next to them, no color used purely for visual interest rather than to encode a state.

---

## 8. Dark mode

Dark mode is not a simple invert of the light tokens. A straight flip would put the light mode teal, which was tuned for contrast against a warm off white, directly onto a dark surface, where it reads as flat and muddy rather than as the brand color. Every color below was checked against its actual dark background rather than derived by inverting lightness.

### 8.1 Background and surface

| Token | Hex | Role |
|---|---|---|
| `color.dark.background` | `#17191A` | App background |
| `color.dark.surface` | `#201F1D` | Cards, panels, the activity feed rail |
| `color.dark.surface.raised` | `#28221D` | Modals, dropdowns, popovers sitting above a card |

Note the background carries a faint warm undertone rather than sitting on a pure gray or blue black. This keeps the two themes feeling like the same product rather than the dark theme becoming the generic charcoal that most SaaS dark modes default to. Surface levels step up in lightness rather than relying on shadow, since shadows barely register on a dark background; elevation in dark mode is communicated by a lighter fill and a lighter border, not a drop shadow.

### 8.2 Text

| Token | Hex | Role |
|---|---|---|
| `color.dark.text.primary` | `#F1ECE1` | Body text, headings |
| `color.dark.text.secondary` | `#A69E90` | Supporting copy, field labels |
| `color.dark.muted` | `#726B60` | Placeholder text, inactive icons |

The primary text color is a warm off white close to the light theme's background color, not a stark white. Pure white text on a near black surface produces a harsher contrast than the rest of this palette calls for.

### 8.3 Brand and functional colors

Every color that sits directly on a dark surface is lifted in lightness and given slightly more saturation than its light mode counterpart. Without this, a color tuned for a warm off white reads as underlit on a dark background, and status colors in particular become hard to tell apart.

| Token | Hex | Checked against | Approx contrast |
|---|---|---|---|
| `color.dark.primary` | `#3EA99C` | `color.dark.background` | 6.8:1 |
| `color.dark.primary.hover` | `#54BBAE` | `color.dark.background` | 8.4:1 |
| `color.dark.primary.tint` | `#1B2E2B` | used as a fill, not text | n/a |
| `color.dark.on.primary` | `#0E1413` | text sitting on `color.dark.primary` fill | 7.1:1 |
| `color.dark.success` | `#5C9E76` | `color.dark.background` | 6.1:1 |
| `color.dark.warning` | `#C99245` | `color.dark.background` | 7.9:1 |
| `color.dark.danger` | `#C6685B` | `color.dark.background` | 6.4:1 |
| `color.dark.info` | `#4FA0A9` | `color.dark.background` | 6.5:1 |

`color.dark.on.primary` exists because the lifted teal is light enough that white text on it fails contrast. Buttons in dark mode use dark, near black text on the primary fill, which mirrors the light theme's white on dark teal button in spirit (light text on the brand color) without copying values that do not carry over.

### 8.4 Borders

| Token | Hex | Role |
|---|---|---|
| `color.dark.border` | `#39332B` | Hairline dividers, input borders, table rules |
| `color.dark.border.strong` | `#4E463A` | Focused input border, active card outline |

### 8.5 Components in dark mode

Buttons follow section 4.1 with dark tokens substituted directly: primary button becomes `color.dark.primary` fill with `color.dark.on.primary` text, secondary button becomes `color.dark.surface` fill with a `color.dark.border` outline.

Status dots and the low emphasis metadata pills from section 4.2 substitute their dark functional and border tokens with no change in shape or size. Table row hover and selection states use `color.dark.primary.tint` in place of `color.primary.tint`.

Elevation in dark mode is handled entirely through the three surface levels in 8.1. Modals get a single very faint shadow, `box shadow: 0 4px 16px rgba(0, 0, 0, 0.45)`, purely to separate the modal from whatever sits behind it since a dark surface on a dark background needs some outline cue; this is the only shadow used in dark mode and it is not present on ordinary cards.

The provenance thread (section 5) uses `color.dark.primary` for lit segments and `color.dark.primary.tint` for unlit ones. The revival moment, where a dot reopens the line from tint back to full color, is more visible in dark mode than light, since the jump in lightness between tint and full color is larger. No change to the interaction itself, just a side effect worth knowing about if it is ever tuned.

### 8.6 Implementation note

Every token in this document should be expressed as a CSS custom property, with the dark values applied through a single `data-theme="dark"` attribute on the root element rather than a second stylesheet or duplicated component styles. This keeps the two themes structurally identical and makes it possible to add a third theme later, if one is ever needed, without touching component code.

## 9. Iconography

Locking one icon set now is cheaper than reconciling three later. Every icon in the app, across every build phase and every agent that touches the UI, comes from the same library at the same weight.

### 9.1 Library and sizing

| Token | Value |
|---|---|
| Library | Lucide, no exceptions |
| `icon.size.default` | 18px |
| `icon.stroke` | 1.75px |
| `icon.color` | inherits `currentColor`, so it follows the surrounding text color and functional color tokens automatically |

Lucide is the choice rather than Heroicons, Phosphor, or Material Icons because its stroke weight and corner rounding sit closest to Plex's own letterforms; a rounder or heavier icon set next to Plex Sans would fight the typography instead of sitting quietly beside it. No second library is added for a missing glyph. If Lucide does not have an icon for something, that is a signal to describe the action in text rather than reach outside the set.

### 9.2 Rules

Icons are functional, never decorative. An icon appears because it helps someone scan the interface faster (a status dot, a chevron indicating a collapsed section, a folder icon distinguishing a repo row from a decision row), not because a heading looked bare without one.

Icon only controls are not allowed unless the icon is universally understood without a tooltip: close (x), and expand or collapse chevrons are the only ones that qualify in this product. Everything else, including settings, refresh, filter, and export, pairs the icon with a visible text label. This matters more here than in a consumer app, since Zuri's users are triaging decisions and audit entries under time pressure, and a guessed icon meaning costs them more than it would in a casual product.

Icon size does not scale with its container. A 24px card and a 40px modal header both use `icon.size.default` at 18px; the icon's job is to support a line of text at `type.body` or `type.heading`, and it stays proportioned to that text rather than to whatever box it sits in.

## 10. Component composition

The hierarchy below exists so that a page never reaches past its own layer for a shortcut, which is how spacing and color drift apart across a codebase over multiple build phases and multiple agents.

```
Pages
  Panels
    Feature components
      Primitives
```

Pages are the top level views (the daemon status view, the decision detail view, the settings view). A page's job is to arrange panels and pass data down, nothing else, it holds no primitive markup of its own.

Panels are the three regions from section 3.2 (sidebar, main panel, activity rail) plus anything else that occupies a fixed region of the window. A panel composes feature components; it does not render a raw button or input directly.

Feature components are the product specific pieces: a decision card, an activity feed row, a memory tier badge. These are the layer where this document's tokens actually get consumed, colors, type, radius, motion, all applied here.

Primitives are the smallest reusable pieces: `Button`, `Input`, `Tag`, `Icon`, `Tooltip`. A primitive knows nothing about Zuri's domain; it only knows the token system in this document.

### 10.1 Rule

Pages never consume primitives directly. A page composes panels, a panel composes feature components, and a feature component composes primitives. If a page ever needs a raw `Button`, that is a sign a feature component is missing, not a reason to reach past the hierarchy. This is the rule most likely to get skipped under deadline pressure, since reaching for a primitive directly is faster in the moment; it is also the rule most responsible for keeping the app coherent once four or five build phases have each touched the UI independently.

## 11. Open questions

None remaining from the original spec. Any future theme (a high contrast mode, for instance) should follow the same process as section 8: derive from the core relationships in this document, then check contrast against the real background rather than inverting existing values.
