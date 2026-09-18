# Changes (uncommitted work)

All items below were added on top of the last commits. Frontend verified with
`vue-tsc` + `vite build`; backend changes are review-checked
(run `go build ./...` to confirm, no Go toolchain was available here).

## 1. Age display in guest mode + print (later replaced, see §3)

- Public share API (`GET /api/public/s/{token}`) now also returns `age` and
  `upcoming_age` per birthday (`internal/web/share_handlers.go`).
- Share page (`#/s/:token`) shows the upcoming age in month-calendar cells
  and in the birthday list (hidden when the birth year is unknown).
- Calendar cells in `CalendarView` and rows in `BirthdayListSidebar` also
  show the upcoming age badge.
- New i18n keys: `common.turnsAge`, `share.showAge`, `share.showAgeHint`
  (en + zh).

## 2. Age display is a toggle (shared + private)

- Share page: 🎂 Age button toggles ages in calendar + list (default on).
- Editor (`CalendarView`): new 🎂 Age button in the header (default on);
  it also controls the sidebar list via a new `show-age` prop on
  `BirthdayListSidebar` (defaults to shown when the prop is absent).
- Active toggle buttons are highlighted blue via a new global
  `button.active` rule (`frontend/src/styles/main.css`).
- The 🖨️ Print button was removed from the editor header
  (`CalendarView.vue`); Ctrl+P printing still uses the print stylesheet.

## 2b. Birth-year toggle in edit and view modes

- New 📆 Year button next to the 🎂 Age button, in both the editor
  (`CalendarView`) and the share page (`ShareView`), default on. It shows the
  birth year as a badge next to each name in the month calendar (and the
  "N more" popup), the birthday list, and the sidebar list (via a new
  `show-year` prop, mirroring `show-age`). Unknown years show nothing.
- Backend calendar responses (`birthdayInCell`, used by the authenticated
  month view and the public share month view) now include `birth_year`, so
  the UI uses the exact stored year instead of deriving it.
- Calendar age badges follow the viewed year: navigating to year+1 shows the
  age they'll turn that year (`viewedYear - birthYear`); the current-year
  view keeps the API's `upcoming_age`, with fallback when the birth year is
  unknown. Same rule in the Export preview for multi-page (year/school)
  ranges; the downloaded PDF already used each page's own year.
- New i18n keys: `share.showYear`, `share.showYearHint`, `common.bornIn`
  (badge tooltip, en + zh).

## 3. Guest mode reuses the Export PDF window (replaces printing)

- New public, no-login PDF endpoints (read-only / pure generation, mounted in
  both full and `--guest-only` mode):
  - `GET /api/public/pdf/fonts` — preset font list
  - `GET /api/public/s/{token}/pdf-settings` — owner's saved PDF design
    (starting point for guests; guests cannot save)
  - `POST /api/public/s/{token}/pdf/export` — generates the PDF scoped
    strictly to the share link's birthdays
- `ExportView.vue` accepts an optional `shareToken` prop (share mode: public
  data, no Save button, Back button to the share page).
- New public route `#/s/:token/export` (allowed through the guest-only
  router guard); the share page's Print button is now an
  Export PDF button opening it.
- Logged-in `guest`-role accounts are still blocked from Export by design
  (route guard + `BlockGuestWrites`).

## 4. Printed-view (PDF) localization fixes

- Weekday header and overflow footnote are rendered in the request language
  instead of hardcoded Chinese (backend uses `calendar.*` / new
  `export.footnote` key; frontend sends `lang` in the PDF range and localizes
  its HTML preview the same way).
- New `export.defaultTitle` key: English UI gets
  `[{month}] {count} birthdays this month`. If the stored title is still the
  untouched legacy Chinese default, non-Chinese UIs show the localized
  default (nothing is auto-saved).
- Backend title variables now match the preview: `{month}`, `{monthShort}`,
  `{monthNum}`, `{count}`, `{year}` (previously only `{month}`/`{count}` were
  replaced in the PDF). Hint text (`export.titleTextHint`) documents all five.

## 5. Settings, tags, and more PDF options

- Settings → My Birthday section has its own Save button (same save action
  as the page button, no more scrolling).
- Tag name length raised from 10 to a default of **20 characters** and made
  configurable: new `BIRTHDAYMEMO_TAG_NAME_MAX_LENGTH` env var (1–100, in
  `compose.yaml`, applied via `config.go` `ApplyEnv`; old `config.json`
  files fall back to 20). Backend create/update validation, the
  `tag.nameLimit` message (`Max {max} …`), the tag modal (maxlength, counter,
  rune-counted), and the DB column (100) all follow the configured value,
  which the frontend reads from `GET /api/settings` (`tag_name_max_length`).
- PDF export: "Show ages in calendar" and "Show birth year in calendar" are two
  independent checkboxes (persisted per user in `pdf_settings.show_age` /
  `pdf_settings.show_birth_year`, auto-migrated, both default off). Cells and
  footnotes render `Alice (35)`, `Alice (1990)`, or `Alice (35 · 1990)` when
  both are on; unknown years show just the name. Preview matches the PDF.
- PDF export: new "School Year (Sep – Aug)" range. It always covers the
  **ongoing** school year (Sep this year → Aug next year, or Sep last
  year → Aug this year before September); the radio label shows the exact
  years (e.g. `2025/2026`). Each of the 12 pages uses its own year, so ages
  and `{year}` titles are correct. Downloads as `birthdays-school.pdf`.

## Files touched

- Backend: `internal/web/{share_handlers,pdf_handlers,tag_handlers,settings_handlers,server}.go`,
  `internal/pdfexport/pdf.go`, `internal/config/config.go`,
  `internal/models/{birthday,user}.go`, `internal/i18n/langs/{en,zh}.json`,
  `compose.yaml`
- Frontend: `src/views/{ShareView,CalendarView,ExportView,SettingsView}.vue`,
  `src/components/{BirthdayListSidebar,TagFormModal}.vue`,
  `src/{api/index.ts,api/types.ts,router/index.ts,styles/main.css}`
