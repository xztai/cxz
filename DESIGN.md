---
name: etheyl
description: A clear, artwork-led interface for creators and their supporters.
colors:
  ink: "#171a24"
  muted: "#626879"
  blue: "#315de8"
  blue-hover: "#244bc5"
  blue-soft: "#eef2ff"
  surface: "#f7f8fc"
  white: "#fff"
  line: "#e4e6ef"
  error: "#ad2738"
  field-line: "#aeb6ca"
  placeholder: "#747b8e"
  outline-line: "#bdc3d3"
typography:
  display:
    fontFamily: "Manrope Variable, sans-serif"
    fontSize: "clamp(38px, 3.2vw, 46px)"
    fontWeight: 800
    lineHeight: 1.12
    letterSpacing: "-0.04em"
  headline:
    fontFamily: "Manrope Variable, sans-serif"
    fontSize: "clamp(30px, 3.3vw, 44px)"
    fontWeight: 800
    lineHeight: 1.18
    letterSpacing: "-0.035em"
  title:
    fontFamily: "Manrope Variable, sans-serif"
    fontSize: "19px"
    fontWeight: 800
    lineHeight: 1.4
  body:
    fontFamily: "Manrope Variable, sans-serif"
    fontSize: "14px"
    fontWeight: 400
    lineHeight: 1.8
  label:
    fontFamily: "Manrope Variable, sans-serif"
    fontSize: "13px"
    fontWeight: 700
  control:
    fontFamily: "Manrope Variable, sans-serif"
    fontSize: "15px"
    fontWeight: 700
    lineHeight: 1.5
  input:
    fontFamily: "Manrope Variable, sans-serif"
    fontSize: "15px"
    fontWeight: 400
    lineHeight: "24px"
  hint:
    fontFamily: "Manrope Variable, sans-serif"
    fontSize: "11px"
    fontWeight: 400
    lineHeight: 1.6
rounded:
  icon-control: "8px"
  field: "9px"
  button: "10px"
  media: "12px"
  panel: "16px"
spacing:
  xs: "8px"
  sm: "12px"
  md: "16px"
  lg: "24px"
  xl: "32px"
  section-gap: "48px"
components:
  button-primary:
    backgroundColor: "{colors.blue}"
    textColor: "{colors.white}"
    typography: "{typography.control}"
    rounded: "{rounded.button}"
    padding: "13px 24px"
  button-primary-hover:
    backgroundColor: "{colors.blue-hover}"
  button-outline:
    backgroundColor: "{colors.white}"
    textColor: "{colors.ink}"
    typography: "{typography.control}"
    rounded: "{rounded.button}"
    padding: "13px 24px"
  button-outline-hover:
    backgroundColor: "{colors.blue-soft}"
  text-field:
    backgroundColor: "{colors.white}"
    textColor: "{colors.ink}"
    typography: "{typography.input}"
    rounded: "{rounded.field}"
    padding: "12px 14px"
  registration-success:
    backgroundColor: "{colors.blue-soft}"
    textColor: "{colors.ink}"
    rounded: "{rounded.panel}"
    padding: "24px"
  creator-page:
    backgroundColor: "{colors.white}"
    rounded: "{rounded.panel}"
    padding: "10px"
---

# Design System: etheyl

## Overview

**Creative North Star: "A place for your work"**

White and cool pale surfaces frame dark Manrope type and direct blue actions. The interface feels familiar and orderly, while original artwork supplies its color and personality. The built landing uses a garden illustration and a ceramic-cat 3D rendering inside an explicitly labeled creator-page demonstration.

Controls are precise and gently rounded. Spacious explanatory sections sit beside a compact registration form; account destinations reuse the same typography, fields and actions at a narrower reading width. Motion is limited to state feedback and navigation to the form.

**Key Characteristics:**

- White and cool-blue surfaces with dark, compact headings.
- Blue for primary actions, focus and subscription-access accents.
- Original illustrative and 3D work presented as demonstration content.
- Outlined fields, modest radii and restrained native interactions.

## Colors

The palette uses a single blue accent against cool neutrals; artwork provides the broader color range.

### Primary

- **Action Blue** (`blue`): primary buttons, selected heading text, icon accents, input carets and keyboard focus.
- **Deep Action Blue** (`blue-hover`): primary-button hover and quieter action links or subscription labels.
- **Pale Blue** (`blue-soft`): closing invitation, registration confirmation and outline-button hover.

### Neutral

- **Ink** (`ink`): headings, labels, field values and primary navigation.
- **Muted Slate** (`muted`): descriptions, helper text, captions and secondary metadata.
- **White** (`white`): page ground, inputs and creator demonstration frame.
- **Cool Surface** (`surface`): the setup section's quieter background.
- **Soft Divider** (`line`): header, content and disclosure separators.
- **Field Stroke** (`field-line`), **Placeholder Slate** (`placeholder`) and **Outline Stroke** (`outline-line`): distinct field and secondary-button boundaries without adding elevation.

Error feedback uses **Error Red** (`error`) for invalid-field borders and accompanying text. It is a semantic state color, rather than a second brand accent.

**The Action Blue Rule.** Blue identifies an action, a focus target, headline emphasis or an access distinction; long explanatory copy uses the neutral text colors.

## Typography

**Display Font:** Manrope Variable, with sans-serif fallback.
**Body Font:** Manrope Variable, with sans-serif fallback.

The font is self-hosted through `@fontsource-variable/manrope`. Heavy, tightly spaced headings establish hierarchy; smaller supporting text has more line spacing. Headings use balanced wrapping. There is no separate display or monospace family in the implemented system.

### Hierarchy

- **Display:** the frontmatter's `display` role is the wide-screen landing headline. Blue may emphasize part of the sentence. Its responsive changes are recorded under Layout.
- **Headline:** the `headline` role identifies major explanatory sections. The closing invitation uses a related scale (`clamp(32px, 3.5vw, 46px)`, weight 800, line-height 1.15).
- **Title:** the `title` role describes setup-step headings. Artwork explanation headings are larger (`29px`, weight 800, line-height 1.3, tracking `-0.025em`) on wide screens.
- **Body:** the `body` role describes explanatory and account introduction copy. Hero supporting copy is larger (`18px`, line-height 1.65) and constrained to a compact measure (`360px`).
- **Label and control:** visible field labels use `label`; actions use `control`; field values use `input`. Hint and error copy use `hint`.

## Layout

The shared content container is centered and capped at 1280px, with 40px gutters per side. At 1000px and below, gutters become 24px; at 640px and below, they become 20px. The supported page floor is 320px. Registration and account panels are capped at 420px where the containing layout permits.

The landing's opening grid pairs a registration offer with an artwork demonstration (`0.91fr 1.09fr`, 48px gap). At 950px and below it becomes one column, capped at 580px, with registration first. The form expands within that column. Explanatory artwork rows use an alternating two-column composition and collapse to one column at 560px. The demonstration's pair of work images also becomes one column at that breakpoint.

The wide headline follows the `display` token. At 1100px and below it becomes 36px; at 950px and below it uses `clamp(40px, 6.3vw, 52px)`; at 560px and below it uses `clamp(36px, 8vw, 44px)`. These are observed overrides, not a continuous type scale.

The setup sequence is three columns until 560px, then a vertical numbered list. FAQ content stacks under its heading at 950px. The footer wraps there as well. The closing action becomes full width at 560px. The sidecar records all observed layout thresholds; their names describe the current rules rather than a framework breakpoint scale.

## Elevation & Depth

Most interface surfaces are flat, separated by cool background changes and fine borders. Depth is concentrated on the creator-page demonstration: its white frame has a soft two-layer shadow (`0 16px 60px -16px #202e5a30, 0 3px 12px #202e5a0c`). The overlapping avatar uses a white border to separate it from the cover. Fields and buttons rely on outline and fill rather than shadows.

**The Artwork Frame Rule.** In the current landing, the raised frame belongs to the creator-page demonstration; surrounding controls and explanatory sections stay visually quiet.

## Shapes

Corners are gently rounded rather than pill-shaped: the frontmatter records the recurring field, button, media and panel radii. Artwork is clipped to its image boundary; large example images retain a 3:2 ratio. The creator avatar is a rounded square with a white border. Small demonstration dots and numbered setup markers are circular.

Icons are inline outline SVGs with a 24-unit viewBox, rounded caps and joins, and a stroke width of 1.7. The default rendered size is 20px, with smaller or larger sizes used for access labels and status symbols. Icons accompany text; form and disclosure controls retain accessible names.

## Components

### Buttons

The primary button is confident without visual bulk. Primary and outline variants share a minimum height (50px), bold type and gently curved corners. The primary fill deepens on hover; the outline button changes its border to blue and receives a pale-blue fill. Both transition background and border color over 0.16s with `ease`. There is no custom active-state transform.

Keyboard focus uses a blue outline (3px, offset 4px). Disabled buttons have reduced opacity (0.65) and a waiting cursor. Registration's action spans the form, with its arrow aligned opposite the label. Header login is a smaller outline variant (minimum height 42px).

### Inputs / Fields

Visible bold labels sit above white outlined fields. Fields have a minimum height (50px), a distinct cool stroke and pale placeholder text. Focus changes the border to blue and adds an outline (2px, offset 1px); invalid fields use the error border alongside explanatory text. Hints sit below the field, and errors replace the associated hint where applicable.

The password field reserves space for an inset visibility button (44px by 42px). Its eye icon changes with visibility, while its accessible label and pressed state communicate the action. Hover adds a pale-blue background and blue icon color.

### Navigation

The header is white with a fine bottom border. The heavy wordmark ends in a blue dot; text navigation uses medium-weight Manrope and turns blue on hover. The login action sits at the right edge. At 640px and below, the header shortens from 88px to 74px, the wordmark becomes smaller, and the setup anchor is hidden while author discovery and login remain visible. The skip link appears on focus.

### Cards / Containers

The creator-page frame is a demonstration container, with cover art, an overlapping avatar, a clear fictional identity and two work previews. Access labels pair text with small check or lock icons; subscription text uses deep blue. The frame and the example captions visibly identify demonstration content.

Successful registration navigates to `/auth/check-email`, a centered account panel capped at 420px. A blue mail symbol precedes “Проверьте почту” and the email address, which can wrap anywhere. The full-width resend action has a 60-second cooldown with a visible countdown underneath; email correction and login are secondary text actions. The same screen follows a login attempt with correct credentials and an unverified email. Email correction expands a labeled email and current-password form in place. The landing's closing invitation focuses and scrolls to its registration email field.

### FAQ

Native `details` and `summary` provide independent disclosure rows separated by fine borders. The summary pairs bold text with a blue SVG plus; opening rotates that symbol by 45 degrees. Answer copy is muted, comfortably spaced and followed by a link where the implemented answer provides one. There is no custom disclosure animation.

Reduced-motion preferences disable transitions and animations and change smooth page scrolling to ordinary scrolling.

## Do's and Don'ts

### Do:

- **Do** reuse the blue action and focus treatments with visible field labels.
- **Do** keep illustration and 3D demonstration content clearly labeled as examples.
- **Do** preserve readable Russian wrapping when columns stack or text grows.
- **Do** retain the native disclosure behavior and reduced-motion handling already implemented.

### Don't:

- **Don't** present the fictional workshop or its artwork as a real creator endorsement.
- **Don't** replace visible field labels or error explanations with icon-only feedback.
- **Don't** remove visible keyboard focus from the existing controls.
