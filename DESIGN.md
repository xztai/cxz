---
name: etheyl
description: A cobalt cultural journal for independent creativity and the people awaiting its continuation.
colors:
  blue: "#2448e5"
  blue-deep: "#1635b8"
  blue-ink: "#dce3ff"
  lemon: "#edfab2"
  orange: "#f5a481"
  ink: "#20251f"
  paper: "#f8f9f5"
  card: "#ffffff"
  muted: "#eceee7"
  muted-foreground: "#62665e"
  border: "#d5d9ce"
  input: "#bec5b6"
  destructive: "#b02d28"
  audience-author: "#e9eddf"
  audience-supporter: "#e7ecfc"
  status: "#f5f7ef"
typography:
  display:
    fontFamily: "Unbounded, sans-serif"
    fontSize: "clamp(46px, 5.9vw, 84px)"
    fontWeight: 600
    lineHeight: 1.17
    letterSpacing: "-0.04em"
  headline:
    fontFamily: "Unbounded, sans-serif"
    fontSize: "36px"
    fontWeight: 500
    lineHeight: 1.3
    letterSpacing: "-0.035em"
  auth-title:
    fontFamily: "Unbounded, sans-serif"
    fontSize: "29px"
    fontWeight: 500
    lineHeight: 1.35
    letterSpacing: "-0.04em"
  title:
    fontFamily: "Golos Text, sans-serif"
    fontSize: "24px"
    fontWeight: 500
    lineHeight: 1.3
    letterSpacing: "-0.025em"
  body:
    fontFamily: "Golos Text, sans-serif"
    fontSize: "16px"
    fontWeight: 400
    lineHeight: 1.6
  body-small:
    fontFamily: "Golos Text, sans-serif"
    fontSize: "14px"
    fontWeight: 400
    lineHeight: 1.75
  action:
    fontFamily: "Golos Text, sans-serif"
    fontSize: "14px"
    fontWeight: 500
    lineHeight: "20px"
  label:
    fontFamily: "Golos Text, sans-serif"
    fontSize: "13px"
    fontWeight: 500
    lineHeight: 1.375
  brand:
    fontFamily: "Unbounded, sans-serif"
    fontSize: "27px"
    fontWeight: 700
    lineHeight: 1
    letterSpacing: "-0.04em"
rounded:
  control: "8px"
  surface: "12px"
  print: "0px"
  circle: "50%"
spacing:
  "8": "8px"
  "12": "12px"
  "16": "16px"
  "18": "18px"
  "20": "20px"
  "22": "22px"
  "24": "24px"
  "28": "28px"
  "32": "32px"
  "40": "40px"
  "48": "48px"
  "60": "60px"
  "64": "64px"
components:
  button-primary:
    backgroundColor: "{colors.blue}"
    textColor: "{colors.card}"
    typography: "{typography.action}"
    rounded: "{rounded.control}"
    padding: "0 16px"
    height: "44px"
  button-primary-hover:
    backgroundColor: "{colors.ink}"
  button-secondary:
    backgroundColor: "{colors.lemon}"
    textColor: "{colors.ink}"
    typography: "{typography.action}"
    rounded: "{rounded.control}"
    padding: "0 16px"
    height: "44px"
  button-outline:
    backgroundColor: "{colors.paper}"
    textColor: "{colors.ink}"
    typography: "{typography.action}"
    rounded: "{rounded.control}"
    padding: "0 16px"
    height: "44px"
  button-link:
    textColor: "{colors.blue}"
    typography: "{typography.action}"
    rounded: "{rounded.control}"
    padding: "0 16px"
    height: "44px"
  auth-submit:
    backgroundColor: "{colors.blue}"
    textColor: "{colors.card}"
    typography: "{typography.action}"
    rounded: "{rounded.control}"
    padding: "0 18px"
    height: "50px"
    width: "100%"
  auth-input:
    backgroundColor: "{colors.card}"
    textColor: "{colors.ink}"
    typography: "{typography.body}"
    rounded: "{rounded.control}"
    padding: "12px 14px"
    height: "48px"
    width: "100%"
  auth-navigation:
    textColor: "{colors.muted-foreground}"
    padding: "0 0 12px"
  auth-navigation-active:
    textColor: "{colors.blue}"
  audience-card-author:
    backgroundColor: "{colors.audience-author}"
    textColor: "{colors.ink}"
    rounded: "{rounded.surface}"
    padding: "34px"
  audience-card-supporter:
    backgroundColor: "{colors.audience-supporter}"
    textColor: "{colors.ink}"
    rounded: "{rounded.surface}"
    padding: "34px"
  journal-piece:
    backgroundColor: "{colors.paper}"
    textColor: "{colors.ink}"
    rounded: "{rounded.print}"
    padding: "12px"
  auth-status:
    backgroundColor: "{colors.status}"
    textColor: "{colors.ink}"
    rounded: "{rounded.control}"
    padding: "8px 10px"
---

# Design System: etheyl

## Overview

**Creative North Star: "The Independent Cultural Journal"**

etheyl speaks through the visual language of an independent cultural journal: expansive cobalt fields, pale yellow emphasis, generous paper margins, and confident Cyrillic headlines. Unbounded gives ideas a distinct public voice; Golos Text keeps explanations, navigation, and account tasks direct and readable. The eight-ray mark and fine orbital drawings connect the identity across surfaces.

Editorial compositions can be expressive and layered. Account tasks become quieter: a white form, visible labels, and a single cobalt submit action. Both expressions belong to the same world through their typography, palette, and carefully controlled geometry. Russian remains the interface language, and etheyl remains the brand name.

**Key Characteristics:**

- Broad cobalt and paper regions with pale yellow emphasis.
- Unbounded headlines paired with practical Golos Text.
- Flat functional surfaces and selectively layered printed material.
- Fine geometric marks, soft control corners, and square journal pieces.
- Responsive simplification that gives account tasks the available space.

## Colors

The palette pairs saturated Cobalt with Pale Lemon and cool paper; Peach belongs to authored creative material.

### Primary

- **Cobalt** (`blue`): large expressive regions, account actions, links, focus, and geometric identity marks.
- **Deep Cobalt** (`blue-deep`): text on the keyboard skip link and selected text.
- **Cobalt Paper Ink** (`blue-ink`): supporting copy and captions on cobalt regions.

### Secondary

- **Pale Lemon** (`lemon`): calls to action on cobalt, selected headline words, an introductory process region, and one journal piece.

### Tertiary

- **Peach** (`orange`): the sound-themed journal piece, giving authored material a second warm paper stock.

### Neutral

- **Green Black Ink** (`ink`): primary reading text and the actual primary-button hover fill.
- **Cool Paper** (`paper`): the surrounding page, neutral buttons, and photographic journal stock.
- **White** (`card`): account form workspace and text on cobalt controls.
- **Muted Paper** (`muted`): quiet hover surfaces for neutral controls.
- **Muted Ink** (`muted-foreground`): supporting account text, secondary navigation, field guidance, and the footer.
- **Paper Rule** (`border`): section rules and neutral control borders.
- **Field Rule** (`input`): text-field borders.
- **Author Paper** (`audience-author`) and **Supporter Paper** (`audience-supporter`): paired, softly tinted audience containers.
- **Status Paper** (`status`): account status callouts.
- **Error Red** (`destructive`): error text, invalid-field borders, and invalid-state rings. This is a semantic state color rather than another brand accent.

### Named Rules

**The Cobalt and Lemon Rule.** Use cobalt as an expressive field or a functional action; use Pale Lemon to hold emphasis against that field. On a white account surface, the principal action is cobalt.

## Typography

**Display Font:** Unbounded, with sans-serif fallback; locally served variable Latin and Cyrillic files.

**Body Font:** Golos Text, with sans-serif fallback; locally served variable Latin and Cyrillic files.

**Character:** Wide, tightly tracked display letters create a recognizable editorial voice. Calm body text carries the practical detail without repeating the display face in every label.

### Hierarchy

- **Display:** the token defines the broad desktop headline. Its responsive overrides use (6vw) below the wide breakpoint, then (clamp(35px, 6.1vw, 49px)) on tablets and (clamp(25px, 6.7vw, 36px)) on phones. Phone line height opens to (1.32).
- **Headline:** section headings use the `headline` role, stepping to (30px) and then (25px) as space narrows. The FAQ uses a smaller sibling at (30px), then (29px) and (24px).
- **Auth title:** the `auth-title` role steps to (28px) on tablets and (25px) on phones. Account instructions remain ordinary Golos Text.
- **Title:** Golos Text headings inside audience containers use `title`, stepping to (21px) and (20px).
- **Body:** the `body` role establishes the reading baseline. Short introductory copy can rise to (19px); compact explanatory paragraphs use `body-small`. Observed copy blocks are kept narrow rather than stretched across the page.
- **Action and label:** the `action` role serves buttons and navigation; `label` serves visible field names. Field input text stays (16px). Account guidance is (11px) with (1.6) line height; footer support copy is (12px).
- **Brand:** the `brand` role accompanies the eight-ray mark and reduces with the header on smaller screens.

### Named Rules

**The Two Voices Rule.** Use Unbounded for the brand and editorial headings; use Golos Text for controls, field labels, explanatory copy, and navigation.

## Layout

The general page inset centers a content span of (1280px), retaining at least (40px) side space. At (800px) and below the inset becomes (24px). Desktop headers are (104px) tall, reducing to (82px) and then (80px). The layout works through broad regions and varied two-column proportions rather than repeating one universal card grid.

Functional groups use compact gaps from the spacing scale. Account fields separate by (18px), each label and control by (8px), with (22px) between form-level groups. The account form is capped at (410px), expanding to a cap of (440px) once its companion region is removed. Desktop authentication uses a (0.95fr / 1.05fr) split in an inset white surface; at (800px) and below it becomes a single form column and the editorial companion is hidden.

The other principal responsive thresholds are (1150px) for reduced gaps and smaller collage scale and (540px) for phone layouts. Paired audience containers become one column on phones. Large explanatory regions use generous vertical spacing, reducing into approximately (50–68px) bands on narrower screens. Expressive collage material follows the introductory action on small screens.

## Elevation & Depth

Functional controls and account surfaces are flat: color regions, fine rules, and spacing establish their hierarchy. The printed creative pieces carry a soft shared shadow and rotated overlap. Their depth belongs to the illusion of loose journal pages; the audience containers and form workspace do not inherit that treatment.

### Shadow Vocabulary

- **Printed piece:** (`box-shadow: 0 14px 32px #0b227930`) under each overlapping creative piece.
- **Field focus:** a (1px) ring in cobalt at half opacity accompanies a cobalt border; invalid fields use a (1px) red ring at one-fifth opacity and a red border. These are state indicators, not surface elevation.

### Named Rules

**The Printed Depth Rule.** Reserve the shared soft shadow and rotated overlap for authored journal material; establish functional hierarchy with flat surfaces and borders.

## Shapes

Controls use the `control` radius and larger bounded surfaces use `surface`. Printed journal pieces keep square corners. Circular step numbers, the status dot, and elliptical line drawings provide a small geometric vocabulary around those rectangles. The brand mark is eight rays drawn as four intersecting strokes; keep its simple line construction.

The split account workspace clips at its outer corners on wide screens. It becomes edge-to-edge and square when reduced to one column. Fine strokes carry dividers, input boundaries, and orbital artwork; there are no hard offset shadows in this delivered world.

## Components

### Buttons

Direct, softly cornered controls with medium-weight Golos Text.

- **Primary:** cobalt fill and white text, using `button-primary`. The actual class-based hover fill is Green Black Ink. Authentication submit buttons use the taller, full-width `auth-submit` variant with the arrow aligned at the far edge.
- **Secondary:** Pale Lemon fill and Green Black Ink text, used against cobalt. Hover mixes (5%) foreground into its fill. The large size is (48px) tall, while the main introductory action is (54px) on desktop and (49px) on phones.
- **Outline:** Cool Paper with a fine Paper Rule border; hover becomes Muted Paper.
- **Link:** cobalt text with an underline on hover.
- **States:** color and border transitions last (200ms); enabled buttons move down (1px) on activation. Disabled controls reduce opacity to (50%) and stop pointer interaction. Button focus uses a (1px) cobalt ring. Ordinary text links use the global (2px) current-color outline, offset by (5px).

### Cards / Containers

Quiet tinted spaces for content, separate from the layered journal material.

- **Audience containers:** Author Paper or Supporter Paper, the `surface` radius, and desktop padding from the component tokens. A fine cobalt drawing sits beside the text; the action is a bottom rule and arrow rather than another filled button.
- **Responsive behavior:** padding steps through (28px) and (25px); symbol and text columns adapt before the pair becomes one column on phones.
- **Account workspace:** white with a bounded form column; the card around the form content adds no independent shadow or border.

### Inputs / Fields

Visible labels and calm white fields support the account task.

- **Style:** `auth-input` with a fine Field Rule border. Placeholders use Muted Ink; the caret is cobalt.
- **Focus / invalid:** use the rings and border changes described in Elevation & Depth. Error text accompanies invalid styling.
- **Password control:** reserve (54px) at the right of the field for a (46px)-wide visibility button and an (18px) eye icon. Its accessible label changes with the visible state.
- **Pending:** the submit action shows a spinner and «Подождите…», and relevant inputs are disabled.

### Navigation

The header pairs the eight-ray brand with simple text links and the account action. Text links underline on hover; explanatory section links disappear below the tablet threshold while account access remains available.

The account navigation is a quiet horizontal rule with two Golos Text links, separated by (28px). The active page uses cobalt text and a (2px) cobalt bottom rule, with `aria-current` carrying the same state. A muted return link sits above it and becomes cobalt on hover.

### Status Callouts

Account status uses Status Paper, the control radius, a fine border, and compact padding. Errors use Error Red text and explicit explanatory copy. Preserve the actual shared status background; do not treat unused destructive-background selectors as an established variant.

### Journal Collage

Three authored pieces overlap in distinct paper colors: a photographic page, a sound study, and a short typographic note. Their project titles are illustrative work, and the collage carries a caption identifying the material as sketches. Keep this distinction when the pattern is reused.

On hover-capable devices, the pages fan slightly over (550ms) using the shared ease-out curve. Wide screens can introduce the collage with a single (1.1s) upward assembly and saturation change; it remains visible throughout. Reduced-motion settings remove animations and transitions and make scrolling immediate.

### Disclosure Rows

FAQ rows use native disclosure behavior, a fine top/bottom rule, medium-weight question text, and a cobalt plus icon. Opening rotates the icon by (45deg); answers sit below in smaller Muted Ink text. The whole summary remains keyboard accessible.

## Do's and Don'ts

### Do:

- **Do** pair cobalt regions with Pale Lemon emphasis and use cobalt actions on white account surfaces.
- **Do** use Unbounded for editorial headings and Golos Text for practical reading and controls.
- **Do** keep visible field labels, readable input text, explicit errors, and visible keyboard focus.
- **Do** reserve rotated overlap and the shared soft shadow for authored journal material.
- **Do** simplify account layouts into one focused column when the companion region no longer fits.
- **Do** identify illustrative project artwork as sketches and distinguish available account actions from planned product capabilities.

### Don't:

- **Don't** carry the replaced visual identity into new surfaces; retain the etheyl name within this cultural-journal world.
- **Don't** turn every content container into a raised or rotated journal piece.
- **Don't** spread the display face into field labels and dense practical copy.
- **Don't** hide core account actions behind the decorative editorial region on small screens.
- **Don't** make entrance motion a prerequisite for seeing content or completing a task.
