/* Bucky x shadcn/ui design prototype: screen composition and interactions */

const icon = (name, className = "") =>
  `<svg class="icon${className ? ` ${className}` : ""}" aria-hidden="true"><use href="#i-${name}"></use></svg>`;

const checkbox = (checked, label) =>
  `<input type="checkbox" class="checkbox"${checked ? " checked" : ""} aria-label="${label}">`;

const badge = (text, kind = "secondary", iconName = "") =>
  `<span class="badge badge-${kind}">${iconName ? icon(iconName) : ""}${text}</span>`;

const chevron = icon("chevron-down", "select-chevron");

function titlebar({ mac = true } = {}) {
  return `<div class="titlebar ${mac ? "mac" : "win"}">
    ${
      mac
        ? `<span class="traffic" aria-hidden="true"><i class="tl-close"></i><i class="tl-min"></i><i class="tl-max"></i></span>`
        : ""
    }
    <div class="brand"><span class="brand-tile">B</span><span class="brand-word">Bucky</span></div>
    ${
      mac
        ? ""
        : `<div class="window-controls">
            <button class="window-control" type="button" aria-label="Minimize">${icon("minus")}</button>
            <button class="window-control" type="button" aria-label="Maximize">${icon("square")}</button>
            <button class="window-control close" type="button" aria-label="Close">${icon("x")}</button>
          </div>`
    }
  </div>`;
}

const connections = [
  { name: "Acme Production", meta: "3 buckets", active: true },
  { name: "Design Sandbox", meta: "2 buckets" },
  { name: "Personal Archive", meta: "Read-only", lock: true },
];

const buckets = [
  { name: "prod-assets", count: "128", active: true },
  { name: "prod-backups" },
  { name: "design-exports" },
];

function sidebar({ empty = false, transfers = 2 } = {}) {
  const connectionRows = connections
    .map(
      (connection) => `<button class="conn${connection.active ? " active" : ""}" type="button" data-goto="workspace">
        <span class="conn-icon">${icon("cloud")}</span>
        <span class="conn-copy">
          <span class="conn-name">${connection.name}</span>
          <span class="conn-meta">${connection.meta}</span>
        </span>
        ${connection.lock ? icon("lock-keyhole", "conn-lock") : '<span class="conn-dot" aria-hidden="true"></span>'}
      </button>`,
    )
    .join("");

  const bucketRows = buckets
    .map(
      (bucket) => `<button class="bucket${bucket.active ? " active" : ""}" type="button" data-goto="workspace">
        ${icon("box")}<span class="bucket-name">${bucket.name}</span>
        ${bucket.count ? `<span class="bucket-count">${bucket.count}</span>` : ""}
      </button>`,
    )
    .join("");

  return `<aside class="sidebar">
    <div class="side-section">
      <div class="side-head"><span>Connections</span><button class="icon-btn" type="button" title="Add connection" aria-label="Add connection" data-goto="connection">${icon("plus")}</button></div>
      ${
        empty
          ? `<div class="side-empty">${icon("cloud-upload")}<span>No connections yet</span><button class="btn btn-outline btn-sm" type="button" data-goto="connection">${icon("plus")}Add connection</button></div>`
          : connectionRows
      }
    </div>
    ${
      empty
        ? ""
        : `<div class="side-section">
            <div class="side-head"><span>Buckets</span><button class="icon-btn" type="button" title="Refresh buckets" aria-label="Refresh buckets">${icon("refresh-cw")}</button></div>
            ${bucketRows}
          </div>`
    }
    <div class="side-spacer"></div>
    <div class="side-foot">
      <button class="side-link" type="button" data-goto="transfers">${icon("arrow-up-down")}<span>Transfers</span>${transfers > 0 ? `<span class="count-badge">${transfers}</span>` : ""}</button>
      <button class="side-link" type="button" data-goto="settings">${icon("settings")}<span>Settings</span><span class="kbd-hint">⌘,</span></button>
    </div>
  </aside>`;
}

/* ------------------------------------------------------------------ */
/* Workspace                                                           */
/* ------------------------------------------------------------------ */

const objects = [
  { name: "campaigns", type: "Folder", hint: "18 objects", size: "—", date: "Oct 14, 2026", kind: "folder" },
  { name: "originals", type: "Folder", hint: "42 objects", size: "—", date: "Oct 11, 2026", kind: "folder" },
  { name: "retouched", type: "Folder", hint: "6 objects", size: "—", date: "Oct 9, 2026", kind: "folder" },
  { name: "hero-aurora.jpg", type: "JPEG image", size: "4.8 MB", date: "Oct 16, 2026", kind: "image", thumb: true, checked: true },
  { name: "launch-film.mp4", type: "MP4 video", size: "128 MB", date: "Oct 15, 2026", kind: "video" },
  { name: "palette-notes.md", type: "Markdown", size: "3.2 KB", date: "Oct 14, 2026", kind: "text" },
  { name: "brand-kit.zip", type: "ZIP archive", size: "62 MB", date: "Oct 10, 2026", kind: "archive", checked: true },
  { name: "session-01.flac", type: "FLAC audio", size: "38 MB", date: "Oct 8, 2026", kind: "audio" },
  { name: "moodboard.pdf", type: "PDF document", size: "12.4 MB", date: "Oct 6, 2026", kind: "pdf" },
  { name: "credits.txt", type: "Text", size: "1.1 KB", date: "Oct 3, 2026", kind: "text" },
  { name: "shot-list.csv", type: "CSV", size: "6.4 KB", date: "Sep 30, 2026", kind: "csv" },
];

const fileIcons = {
  folder: "folder",
  image: "file-image",
  video: "file-video",
  audio: "file-audio",
  text: "file-text",
  archive: "file-archive",
  pdf: "file-text",
  csv: "file-text",
};

function fileTile(object) {
  if (object.thumb) return '<span class="file-tile file-thumb" aria-hidden="true"></span>';
  return `<span class="file-tile ${object.kind === "folder" ? "folder" : ""}" aria-hidden="true">${icon(fileIcons[object.kind])}</span>`;
}

function objectRow(object) {
  return `<tr class="object-row${object.checked ? " selected" : ""}${object.kind === "folder" ? " is-folder" : ""}"${object.name === "hero-aurora.jpg" ? ' data-goto="preview" tabindex="0"' : ""}>
    <td class="check-col">${checkbox(Boolean(object.checked), `Select ${object.name}`)}</td>
    <td><div class="name-cell">${fileTile(object)}<span class="name-copy"><span class="file-name">${object.name}</span>${object.hint ? `<span class="file-hint">${object.hint}</span>` : ""}</span></div></td>
    <td class="col-type type-cell">${object.type}</td>
    <td class="num size-cell">${object.size}</td>
    <td class="col-date date-cell">${object.date}</td>
    <td><button class="row-actions" type="button" aria-label="Actions for ${object.name}" tabindex="-1">${icon("more-horizontal")}</button></td>
  </tr>`;
}

function gridCard(object) {
  const art = object.thumb ? "aurora" : object.kind === "folder" ? "folder" : "";
  return `<div class="object-card${object.checked ? " selected" : ""}" role="button" tabindex="0" data-goto="${object.name === "hero-aurora.jpg" ? "preview" : "workspace"}">
    <div class="object-preview ${art}">${object.thumb ? "" : icon(fileIcons[object.kind] || "file")}</div>
    <div class="object-meta">
      <div class="object-name">${object.name}</div>
      <div class="object-sub"><span>${object.type}</span><span>${object.size}</span></div>
    </div>
  </div>`;
}

function pageHead() {
  return `<header class="page-head">
    <div>
      <div class="eyebrow"><span class="status-dot" aria-hidden="true"></span>Acme Production</div>
      <h1>prod-assets</h1>
    </div>
    <div class="page-actions">
      <label class="searchbox">${icon("search")}<input type="search" placeholder="Search objects…" aria-label="Search objects"><kbd>⌘K</kbd></label>
      <button class="btn btn-outline btn-icon" type="button" title="Refresh objects" aria-label="Refresh objects">${icon("refresh-cw")}</button>
      <div class="split-button">
        <button class="btn btn-primary" type="button">${icon("upload")}Upload</button>
        <button class="btn btn-primary" type="button" aria-label="Upload options">${icon("chevron-down")}</button>
      </div>
    </div>
  </header>`;
}

function tableCard() {
  return `<div class="table-card">
    <div class="table-toolbar">
      <div class="crumbs" aria-label="Breadcrumb">
        <button class="crumb" type="button">${icon("box")}All objects</button>
        ${icon("chevron-right", "crumb-sep")}
        <button class="crumb" type="button">photos</button>
        ${icon("chevron-right", "crumb-sep")}
        <button class="crumb current" type="button" aria-current="page">2026</button>
      </div>
      <div class="toolbar-right">
        <span class="meta-count">11 of 128 objects · 2.4 GB</span>
        <span class="toolbar-sep" aria-hidden="true"></span>
        <button class="sort-btn" type="button">${icon("arrow-up-down")}Last modified${icon("chevron-down")}</button>
        <div class="view-toggle" role="group" aria-label="View">
          <button class="on" type="button" data-view="list" title="List view" aria-label="List view" aria-pressed="true">${icon("list")}</button>
          <button type="button" data-view="grid" title="Grid view" aria-label="Grid view" aria-pressed="false">${icon("grid-3x3")}</button>
        </div>
      </div>
    </div>
    <div class="table-scroll">
      <table>
        <colgroup>
          <col style="width: 38px">
          <col>
          <col style="width: 128px">
          <col style="width: 84px">
          <col style="width: 112px">
          <col style="width: 38px">
        </colgroup>
        <thead>
          <tr>
            <th class="check-col"><input type="checkbox" class="checkbox" aria-label="Select all objects" data-select-all></th>
            <th>Name</th>
            <th class="col-type">Type</th>
            <th class="num">Size</th>
            <th class="col-date">Modified</th>
            <th></th>
          </tr>
        </thead>
        <tbody>${objects.map(objectRow).join("")}</tbody>
      </table>
    </div>
    <div class="object-grid" aria-label="Objects, grid view">
      ${objects.slice(0, 8).map(gridCard).join("")}
    </div>
    <div class="table-foot">
      <span>Showing 11 of 128 objects</span>
      <button class="load-more" type="button">${icon("arrow-down-to-line")}Load more</button>
    </div>
  </div>`;
}

function selectionBar() {
  return `<div class="selection-bar" data-selection-bar>
    <span class="selection-count" data-selection-count>2 selected</span>
    <span class="selection-sep" aria-hidden="true"></span>
    <button class="btn" type="button">${icon("download")}Download</button>
    <button class="btn" type="button">${icon("share-2")}Share</button>
    <button class="btn" type="button">${icon("folder-up")}Move</button>
    <button class="btn danger" type="button" data-goto="delete">${icon("trash-2")}Delete</button>
    <span class="selection-sep" aria-hidden="true"></span>
    <button class="selection-clear" type="button" aria-label="Clear selection" data-clear-selection>${icon("x")}</button>
  </div>`;
}

function workspaceApp({ overlay = "" } = {}) {
  return `<div class="app">
    ${titlebar()}
    <div class="app-body">
      ${sidebar()}
      <main class="main">
        ${pageHead()}
        ${tableCard()}
        ${selectionBar()}
      </main>
    </div>
    ${overlay}
  </div>`;
}

function welcomeApp() {
  return `<div class="app">
    ${titlebar()}
    <div class="app-body">
      ${sidebar({ empty: true, transfers: 0 })}
      <main class="main">
        <div class="welcome">
          <div class="welcome-inner">
            <div class="welcome-mark">${icon("cloud-upload")}</div>
            <div class="eyebrow">Welcome to Bucky</div>
            <h1 class="welcome-title">Your R2 buckets,<br><em>without the noise.</em></h1>
            <p class="welcome-copy">A calm, local-first home for Cloudflare R2. Browse, preview and move objects without leaving your desktop.</p>
            <div class="welcome-actions">
              <button class="btn btn-primary btn-lg" type="button" data-goto="connection">${icon("plus")}Add R2 connection</button>
            </div>
            <div class="welcome-hint">${icon("lock-keyhole")}Secrets stay in your system keychain, never in Bucky's database.</div>
            <div class="welcome-features">
              <div>
                <span class="welcome-feature-icon">${icon("folder-open")}</span>
                <h3>Browse &amp; preview</h3>
                <p>Move through prefixes and preview images, text and media in place.</p>
              </div>
              <div>
                <span class="welcome-feature-icon">${icon("link")}</span>
                <h3>Links on your terms</h3>
                <p>Create expiring pre-signed links for a single object, never a folder.</p>
              </div>
              <div>
                <span class="welcome-feature-icon">${icon("rotate-cw")}</span>
                <h3>Transfers that resume</h3>
                <p>Pause, resume and recover large uploads after an interruption.</p>
              </div>
            </div>
          </div>
        </div>
      </main>
    </div>
  </div>`;
}

/* ------------------------------------------------------------------ */
/* Connection dialog                                                   */
/* ------------------------------------------------------------------ */

function connectionOverlay() {
  return `<div class="scrim" data-goto="workspace"></div>
  <div class="dialog" role="dialog" aria-modal="true" aria-labelledby="connection-title">
    <header class="dialog-head">
      <span class="dialog-icon">${icon("cloud-upload")}</span>
      <div>
        <h2 id="connection-title">New R2 connection</h2>
        <p>Connect a Cloudflare R2 bucket. Credentials stay on this device.</p>
      </div>
      <button class="icon-btn dialog-close" type="button" aria-label="Close" data-goto="workspace">${icon("x")}</button>
    </header>
    <div class="dialog-body">
      <div class="field">
        <label for="connection-name">Connection name</label>
        <input class="input" id="connection-name" value="Acme Production">
      </div>
      <div class="form-row">
        <div class="field">
          <label for="account-id">Account ID</label>
          <input class="input mono" id="account-id" value="9f2ce4a1b7d84c53">
        </div>
        <div class="field">
          <label for="jurisdiction">Jurisdiction</label>
          <div class="select-wrap">
            <select class="select" id="jurisdiction"><option>Automatic</option><option>European Union</option><option>FedRAMP</option></select>
            ${chevron}
          </div>
        </div>
      </div>
      <div class="form-row">
        <div class="field">
          <label for="bucket">Bucket <span class="optional">Optional</span></label>
          <input class="input" id="bucket" value="prod-assets">
        </div>
        <div class="field">
          <label for="public-url">Public URL <span class="optional">Optional</span></label>
          <input class="input" id="public-url" placeholder="https://cdn.acme.studio">
        </div>
      </div>
      <div class="field">
        <span class="field-label">Credentials</span>
        <div class="segmented" role="group" aria-label="Credential source">
          <button class="on" type="button">${icon("key-round")}System keychain</button>
          <button type="button">${icon("command")}Environment</button>
        </div>
      </div>
      <div class="form-row">
        <div class="field">
          <label for="access-key">Access key ID</label>
          <input class="input mono" id="access-key" value="R2AK7F3C91D04E8A6">
        </div>
        <div class="field">
          <label for="secret-key">Secret access key</label>
          <div class="input-wrap">
            <input class="input mono" id="secret-key" type="password" value="bucky-placeholder-secret">
            <button class="input-affix" type="button" aria-label="Show secret key">${icon("eye")}</button>
          </div>
        </div>
      </div>
      <div class="switch-row">
        <button class="switch" type="button" role="switch" aria-checked="false" aria-label="Read-only connection"></button>
        <span class="switch-copy">
          <strong>Read-only connection</strong>
          <small>Block uploads, moves and deletes for this connection.</small>
        </span>
      </div>
      <div class="notice">${icon("shield-check")}<span>Your secret key is stored in the system keychain and never written to Bucky's database.</span></div>
    </div>
    <footer class="dialog-foot">
      <span class="foot-note">${icon("lock-keyhole")}Encrypted at rest</span>
      <div class="dialog-foot-actions">
        <button class="btn btn-ghost" type="button" data-goto="workspace">Cancel</button>
        <button class="btn btn-primary" type="button">Save connection</button>
      </div>
    </footer>
  </div>`;
}

/* ------------------------------------------------------------------ */
/* Preview sheet                                                       */
/* ------------------------------------------------------------------ */

function previewOverlay() {
  return `<div class="sheet-scrim" data-goto="workspace"></div>
  <aside class="sheet" role="dialog" aria-modal="true" aria-labelledby="preview-title">
    <header class="sheet-head">
      <h2 id="preview-title">Object details</h2>
      <span class="badge badge-outline">JPEG</span>
      <button class="icon-btn sheet-close" type="button" aria-label="Close" data-goto="workspace">${icon("x")}</button>
    </header>
    <div class="sheet-body">
      <div class="preview-media"><span class="media-badge">4096 × 2731</span></div>
      <div class="preview-content">
        <div class="preview-title-row">
          <div>
            <h3 class="preview-title">hero-aurora.jpg</h3>
            <p class="preview-sub">Image · Updated 2 days ago</p>
          </div>
          <div class="preview-actions">
            <button class="btn btn-primary" type="button">${icon("download")}Download</button>
            <button class="btn btn-outline btn-icon" type="button" title="Share" aria-label="Share">${icon("share-2")}</button>
            <button class="btn btn-ghost btn-icon" type="button" title="More actions" aria-label="More actions">${icon("more-horizontal")}</button>
          </div>
        </div>
        <div class="key-box">${icon("link")}<span>photos/2026/hero-aurora.jpg</span><button class="icon-btn" type="button" aria-label="Copy object key">${icon("copy")}</button></div>
        <div class="meta-grid">
          <div><div class="meta-label">Size</div><div class="meta-value">4.8 MB</div></div>
          <div><div class="meta-label">Dimensions</div><div class="meta-value">4096 × 2731</div></div>
          <div><div class="meta-label">Modified</div><div class="meta-value">Oct 16, 2026</div></div>
          <div><div class="meta-label">Uploaded</div><div class="meta-value">Oct 12, 2026</div></div>
        </div>
        <div class="tabs" role="tablist" aria-label="Object information">
          <button class="on" type="button" role="tab" aria-selected="true">Details</button>
          <button type="button" role="tab" aria-selected="false">Metadata</button>
        </div>
        <dl class="meta-list">
          <div class="meta-row"><dt>Content-Type</dt><dd>image/jpeg</dd></div>
          <div class="meta-row"><dt>ETag</dt><dd class="mono">9f8c2b6d4a1e…e41a</dd></div>
          <div class="meta-row"><dt>Cache-Control</dt><dd>public, max-age=31536000</dd></div>
          <div class="meta-row"><dt>Storage class</dt><dd>Standard</dd></div>
        </dl>
      </div>
    </div>
    <footer class="sheet-foot">
      <button class="link-btn danger" type="button">${icon("trash-2")}Delete object</button>
      <button class="btn btn-ghost btn-sm" type="button" data-goto="workspace">Close</button>
    </footer>
  </aside>`;
}

/* ------------------------------------------------------------------ */
/* Transfers sheet                                                     */
/* ------------------------------------------------------------------ */

function transferItem({ kind, name, state, percent, detail, speed, actions }) {
  return `<div class="transfer${state === "done" ? " done" : ""}">
    <span class="transfer-icon">${icon(state === "done" ? "check" : kind)}</span>
    <div class="transfer-main">
      <div class="transfer-top"><span class="transfer-name">${name}</span>${state === "done" ? "" : `<span class="transfer-pct">${percent}%</span>`}</div>
      ${state === "done" ? "" : `<div class="progress" aria-hidden="true"><i style="width:${percent}%"></i></div>`}
      <div class="transfer-meta"><span>${detail}</span>${speed ? `<span class="dot"></span><span>${speed}</span>` : ""}</div>
    </div>
    <div class="transfer-actions">${actions}</div>
  </div>`;
}

function transfersOverlay() {
  return `<div class="sheet-scrim" data-goto="workspace"></div>
  <aside class="sheet" role="dialog" aria-modal="true" aria-labelledby="transfers-title">
    <header class="sheet-head">
      <h2 id="transfers-title">Transfers</h2>
      <span class="badge badge-accent">2 active</span>
      <button class="icon-btn sheet-close" type="button" aria-label="Close" data-goto="workspace">${icon("x")}</button>
    </header>
    <div class="sheet-body">
      <div class="transfer-tabs">
        <div class="tabs full" role="tablist" aria-label="Transfer filter">
          <button class="on" type="button" role="tab" aria-selected="true">All <span class="count-pill">4</span></button>
          <button type="button" role="tab" aria-selected="false">Active <span class="count-pill">2</span></button>
          <button type="button" role="tab" aria-selected="false">Completed <span class="count-pill">2</span></button>
        </div>
      </div>
      <div class="transfer-group-label">Active <span>2</span></div>
      ${transferItem({
        kind: "file-video",
        name: "launch-film.mp4",
        state: "active",
        percent: 66,
        detail: "Upload · 84.2 of 128 MB",
        speed: "12.4 MB/s · 4s left",
        actions: `<button class="icon-btn" type="button" aria-label="Pause transfer">${icon("pause")}</button><button class="icon-btn" type="button" aria-label="Cancel transfer">${icon("x")}</button>`,
      })}
      ${transferItem({
        kind: "file-archive",
        name: "product-shots.zip",
        state: "active",
        percent: 43,
        detail: "Download · 26.6 of 62 MB",
        speed: "8.1 MB/s · 5s left",
        actions: `<button class="icon-btn" type="button" aria-label="Pause transfer">${icon("pause")}</button><button class="icon-btn" type="button" aria-label="Cancel transfer">${icon("x")}</button>`,
      })}
      <div class="transfer-group-label">Completed <span>2</span></div>
      ${transferItem({
        kind: "file-image",
        name: "hero-aurora.jpg",
        state: "done",
        detail: "Uploaded · 4.8 MB · 2 min ago",
        actions: `<button class="icon-btn" type="button" aria-label="Show file">${icon("folder-open")}</button>`,
      })}
      ${transferItem({
        kind: "file-text",
        name: "moodboard.pdf",
        state: "done",
        detail: "Downloaded · 12.4 MB · 18 min ago",
        actions: `<button class="icon-btn" type="button" aria-label="Show file">${icon("folder-open")}</button>`,
      })}
      <div class="transfer-drop">${icon("upload")}<span>Drop files to upload to <strong>photos/2026</strong></span></div>
    </div>
    <footer class="sheet-foot">
      <div class="speed-row">${icon("gauge")}Speed limit
        <div class="select-wrap"><select class="select" aria-label="Speed limit"><option>Unlimited</option><option>20 MB/s</option><option>5 MB/s</option></select>${chevron}</div>
      </div>
      <button class="link-btn" type="button" data-goto="settings">${icon("sliders-horizontal")}Settings</button>
    </footer>
  </aside>`;
}

/* ------------------------------------------------------------------ */
/* Delete plan dialog                                                  */
/* ------------------------------------------------------------------ */

function deleteOverlay() {
  return `<div class="scrim" data-goto="workspace"></div>
  <div class="dialog delete" role="alertdialog" aria-modal="true" aria-labelledby="delete-title">
    <header class="dialog-head">
      <span class="dialog-icon danger">${icon("triangle-alert")}</span>
      <div>
        <h2 id="delete-title">Delete 3 objects?</h2>
        <p>These objects will be permanently removed from prod-assets.</p>
      </div>
      <button class="icon-btn dialog-close" type="button" aria-label="Close" data-goto="workspace">${icon("x")}</button>
    </header>
    <div class="dialog-body">
      <div class="plan-card">
        <div class="plan-row"><span class="file-tile">${icon("file-archive")}</span><span class="plan-name">brand-kit.zip</span><span class="plan-size">62 MB</span></div>
        <div class="plan-row"><span class="file-tile">${icon("file-text")}</span><span class="plan-name">moodboard.pdf</span><span class="plan-size">12.4 MB</span></div>
        <div class="plan-row"><span class="file-tile">${icon("file-audio")}</span><span class="plan-name">session-01.flac</span><span class="plan-size">38 MB</span></div>
        <div class="plan-total"><span>3 objects</span><strong>112.4 MB</strong></div>
      </div>
      <div class="plan-meta">${icon("clock")}Plan OP-2481 expires in 9:42<span class="sep"></span><span>Checked again before deletion</span></div>
      <div class="notice danger">${icon("triangle-alert")}<span><strong>This cannot be undone.</strong> Versioning is not enabled on this bucket, so no previous copy can be restored.</span></div>
    </div>
    <footer class="dialog-foot">
      <span class="foot-note">${icon("shield-check")}Per-object results will be recorded</span>
      <div class="dialog-foot-actions">
        <button class="btn btn-ghost" type="button" data-goto="workspace">Cancel</button>
        <button class="btn btn-destructive" type="button">Delete 3 objects</button>
      </div>
    </footer>
  </div>`;
}

/* ------------------------------------------------------------------ */
/* Settings dialog                                                     */
/* ------------------------------------------------------------------ */

function themeCard(mode, label, iconName, previewClass) {
  return `<button class="theme-card" type="button" data-theme-set="${mode}">
    <span class="theme-preview ${previewClass}">
      <span class="tp-bar"></span><span class="tp-side"></span><span class="tp-block"></span><span class="tp-block short"></span>
    </span>
    <span class="theme-label"><span>${icon(iconName)}${label}</span><span class="theme-check">${icon("check")}</span></span>
  </button>`;
}

function settingsOverlay() {
  return `<div class="scrim" data-goto="workspace"></div>
  <div class="dialog settings" role="dialog" aria-modal="true" aria-labelledby="settings-title">
    <header class="dialog-head">
      <div>
        <h2 id="settings-title">Settings</h2>
        <p>Preferences are saved on this device.</p>
      </div>
      <button class="icon-btn dialog-close" type="button" aria-label="Close" data-goto="workspace">${icon("x")}</button>
    </header>
    <div class="settings-body">
      <nav class="settings-nav" aria-label="Settings sections">
        <button class="on" type="button" data-scroll="settings-appearance">${icon("palette")}Appearance</button>
        <button type="button" data-scroll="settings-network">${icon("gauge")}Network</button>
        <button type="button" data-scroll="settings-usage">${icon("database")}Usage</button>
        <div class="nav-spacer"></div>
        <span class="version">Bucky 0.1.0</span>
      </nav>
      <div class="settings-content">
        <section class="settings-section" id="settings-appearance">
          <h3>Theme</h3>
          <p class="section-desc">Choose how Bucky looks. System follows your OS appearance.</p>
          <div class="theme-grid">
            ${themeCard("light", "Light", "sun", "light")}
            ${themeCard("dark", "Dark", "moon", "dark")}
            ${themeCard("system", "System", "monitor", "split")}
          </div>
        </section>
        <section class="settings-section">
          <h3>Language</h3>
          <div class="setting-row">
            <div>
              <div class="field-label">Interface language</div>
              <div class="field-hint">Applies immediately across the app.</div>
            </div>
            <div class="select-wrap">
              <select class="select" aria-label="Interface language"><option>English</option><option>简体中文</option></select>
              ${chevron}
            </div>
          </div>
        </section>
        <section class="settings-section" id="settings-network">
          <h3>Network</h3>
          <p class="section-desc">Limits apply to every active transfer.</p>
          <div class="setting-row">
            <div>
              <div class="field-label">Speed limit</div>
              <div class="field-hint">0 means no limit.</div>
            </div>
            <div class="select-wrap">
              <select class="select" aria-label="Speed limit"><option>Unlimited</option><option>20 MB/s</option><option>5 MB/s</option></select>
              ${chevron}
            </div>
          </div>
          <div class="setting-row">
            <div>
              <div class="field-label">Pre-signed link lifetime</div>
              <div class="field-hint">R2 accepts up to 7 days.</div>
            </div>
            <div class="select-wrap">
              <select class="select" aria-label="Pre-signed link lifetime"><option>1 hour</option><option>24 hours</option><option>7 days</option></select>
              ${chevron}
            </div>
          </div>
        </section>
        <section class="settings-section" id="settings-usage">
          <h3>Usage</h3>
          <p class="section-desc">Last scanned 2 minutes ago across prod-assets.</p>
          <div class="usage-stats">
            <div class="usage-stat"><span class="meta-label">Objects</span><strong>1,284</strong></div>
            <div class="usage-stat"><span class="meta-label">Total size</span><strong>2.4 GB</strong></div>
            <div class="usage-stat"><span class="meta-label">Last scan</span><strong>2 min</strong></div>
          </div>
          <div class="usage-bar" aria-hidden="true">
            <i style="width:58%;background:var(--primary)"></i>
            <i style="width:27%;background:color-mix(in oklab, var(--primary) 42%, var(--muted))"></i>
            <i style="width:15%;background:var(--border)"></i>
          </div>
          <div class="usage-legend">
            <span><i style="background:var(--primary)"></i>Images 58%</span>
            <span><i style="background:color-mix(in oklab, var(--primary) 42%, var(--muted))"></i>Video 27%</span>
            <span><i style="background:var(--border)"></i>Other 15%</span>
          </div>
          <div class="ds-row" style="margin-top: 12px">
            <span class="badge badge-success">Scan complete</span>
            <button class="link-btn" type="button">${icon("refresh-cw")}Rescan</button>
          </div>
        </section>
      </div>
    </div>
    <footer class="dialog-foot">
      <span class="foot-note">${icon("check")}Changes save automatically</span>
      <div class="dialog-foot-actions">
        <button class="btn btn-primary" type="button" data-goto="workspace">Done</button>
      </div>
    </footer>
  </div>`;
}

/* ------------------------------------------------------------------ */
/* Design system board                                                 */
/* ------------------------------------------------------------------ */

function swatch(token) {
  return `<div class="swatch">
    <div class="swatch-color" style="--swatch: var(${token})"></div>
    <div class="swatch-info"><code>${token}</code><span data-token="${token}"></span></div>
  </div>`;
}

function dsButtons() {
  return `<div class="ds-card">
    <h3>Buttons</h3>
    <div class="ds-row">
      <button class="btn btn-primary" type="button">${icon("upload")}Upload</button>
      <button class="btn btn-secondary" type="button">Secondary</button>
      <button class="btn btn-outline" type="button">Outline</button>
      <button class="btn btn-ghost" type="button">Ghost</button>
      <button class="btn btn-destructive" type="button">${icon("trash-2")}Delete</button>
    </div>
    <div class="ds-row">
      <button class="btn btn-outline btn-sm" type="button">Small</button>
      <button class="btn btn-outline btn-icon" type="button" aria-label="Refresh">${icon("refresh-cw")}</button>
      <button class="btn btn-outline btn-icon btn-sm" type="button" aria-label="More">${icon("more-horizontal")}</button>
      <div class="split-button">
        <button class="btn btn-primary btn-sm" type="button">${icon("upload")}Upload</button>
        <button class="btn btn-primary btn-sm" type="button" aria-label="Upload options">${icon("chevron-down")}</button>
      </div>
    </div>
  </div>`;
}

function dsForms() {
  return `<div class="ds-card">
    <h3>Form controls</h3>
    <div class="ds-stack">
      <input class="input" placeholder="Connection name" aria-label="Example input">
      <div class="select-wrap"><select class="select" aria-label="Example select"><option>Automatic</option><option>European Union</option></select>${chevron}</div>
      <div>
        <input class="input is-error" value="9f2ce4" aria-label="Example invalid input" aria-invalid="true">
        <p class="field-hint" style="margin-top: 5px; color: var(--destructive)">Account ID must be 32 characters.</p>
      </div>
    </div>
  </div>`;
}

function dsSelection() {
  return `<div class="ds-card">
    <h3>Selection &amp; status</h3>
    <div class="ds-row">
      <label class="ds-check">${checkbox(true, "Checked example")}Checked</label>
      <label class="ds-check">${checkbox(false, "Unchecked example")}Unchecked</label>
    </div>
    <div class="ds-row">
      <span class="ds-check"><span class="radio on"></span>Selected</span>
      <span class="ds-check"><span class="radio"></span>Unselected</span>
      <button class="switch on" type="button" role="switch" aria-checked="true" aria-label="Enabled example"></button>
      <button class="switch" type="button" role="switch" aria-checked="false" aria-label="Disabled example"></button>
    </div>
    <div class="ds-row">
      ${badge("Default")}${badge("Secondary", "secondary")}${badge("Outline", "outline")}${badge("Complete", "success", "check")}${badge("Warning", "warning")}${badge("Danger", "destructive")}
    </div>
  </div>`;
}

function dsNavigation() {
  return `<div class="ds-card">
    <h3>Navigation</h3>
    <div class="ds-stack">
      <div class="tabs" role="tablist" aria-label="Example tabs">
        <button class="on" type="button">Details</button>
        <button type="button">Metadata</button>
      </div>
      <div class="crumbs">
        <button class="crumb" type="button">${icon("box")}prod-assets</button>
        ${icon("chevron-right", "crumb-sep")}
        <button class="crumb" type="button">photos</button>
        ${icon("chevron-right", "crumb-sep")}
        <button class="crumb current" type="button">2026</button>
      </div>
      <div class="ds-row">
        <div class="view-toggle" role="group" aria-label="Example view toggle">
          <button class="on" type="button" aria-label="List">${icon("list")}</button>
          <button type="button" aria-label="Grid">${icon("grid-3x3")}</button>
        </div>
        <span class="badge badge-outline">${icon("folder")}photos/2026</span>
      </div>
    </div>
  </div>`;
}

function dsFeedback() {
  return `<div class="ds-card">
    <h3>Feedback</h3>
    <div class="ds-stack">
      <div>
        <div class="transfer-meta" style="margin-bottom: 6px"><span>Uploading launch-film.mp4</span><span style="margin-left:auto">66%</span></div>
        <div class="progress" style="margin: 0"><i style="width: 66%"></i></div>
      </div>
      <div>
        <div class="transfer-meta" style="margin-bottom: 6px"><span>Verifying checksum</span><span style="margin-left:auto">100%</span></div>
        <div class="progress done" style="margin: 0"><i style="width: 100%"></i></div>
      </div>
      <div class="notice" style="margin: 0">${icon("shield-check")}<span>Credentials are stored in the system keychain.</span></div>
      <div class="notice danger" style="margin: 0">${icon("triangle-alert")}<span>This action cannot be undone.</span></div>
    </div>
  </div>`;
}

function dsOverlays() {
  return `<div class="ds-card">
    <h3>Menu &amp; tooltip</h3>
    <div class="ds-row" style="align-items: flex-start; gap: 18px">
      <div class="menu">
        <div class="menu-label">Object</div>
        <button class="menu-item" type="button">${icon("download")}Download<span class="shortcut">⌘D</span></button>
        <button class="menu-item" type="button">${icon("share-2")}Create link<span class="shortcut">⌘L</span></button>
        <button class="menu-item" type="button">${icon("copy")}Copy key</button>
        <div class="menu-sep"></div>
        <button class="menu-item danger" type="button">${icon("trash-2")}Delete…</button>
      </div>
      <div class="tooltip-wrap">
        <span class="tooltip">Copy object key</span>
        <button class="btn btn-outline btn-sm" type="button">${icon("copy")}Copy</button>
      </div>
    </div>
  </div>`;
}

function designSystemBoard() {
  return `<div class="board">
    <header class="board-head">
      <div>
        <h1>Bucky design system</h1>
        <p>Warm neutrals, a single terracotta accent and compact desktop density. Built on shadcn/ui primitives, CSS variable tokens and lucide icons.</p>
      </div>
      <div class="board-meta">${badge("Light + dark", "outline")}${badge("8 screens", "accent")}</div>
    </header>

    <section class="board-section">
      <h2>Color tokens</h2>
      <div class="swatch-grid">
        ${swatch("--background")}${swatch("--card")}${swatch("--primary")}${swatch("--accent")}${swatch("--muted")}
        ${swatch("--foreground")}${swatch("--destructive")}${swatch("--success")}${swatch("--border")}${swatch("--sidebar")}
      </div>
    </section>

    <section class="board-section">
      <h2>Typography</h2>
      <div class="type-list">
        <div class="type-row"><span class="type-token">serif / display</span><span class="type-sample-display">A quieter corner of the cloud.</span></div>
        <div class="type-row"><span class="type-token">sans / page title</span><span class="type-sample-title">prod-assets</span></div>
        <div class="type-row"><span class="type-token">sans / section</span><span class="type-sample-section">Object details</span></div>
        <div class="type-row"><span class="type-token">sans / body</span><span class="type-sample-body">Browse, preview and move objects without leaving your desktop.</span></div>
        <div class="type-row"><span class="type-token">sans / small</span><span class="type-sample-small">Updated 2 days ago · 4.8 MB</span></div>
        <div class="type-row"><span class="type-token">sans / label</span><span class="type-sample-label">Connections</span></div>
        <div class="type-row"><span class="type-token">mono / key</span><span class="type-sample-mono">photos/2026/hero-aurora.jpg</span></div>
      </div>
    </section>

    <section class="board-section">
      <h2>Shape &amp; spacing</h2>
      <div class="ds-two-col">
        <div class="ds-card">
          <h3>Radius</h3>
          <div class="radius-row">
            ${[4, 6, 8, 10, 12, 16].map((radius) => `<span class="radius-item"><span class="radius-box" style="border-radius:${radius}px"></span>${radius}</span>`).join("")}
          </div>
        </div>
        <div class="ds-card">
          <h3>Spacing scale</h3>
          <div class="spacing-row">
            ${[4, 8, 12, 16, 24, 32].map((space) => `<span class="spacing-item"><span class="spacing-bar" style="height:${space}px"></span>${space}</span>`).join("")}
          </div>
        </div>
      </div>
    </section>

    <section class="board-section">
      <h2>Components</h2>
      <div class="ds-two-col">
        ${dsButtons()}
        ${dsForms()}
        ${dsSelection()}
        ${dsNavigation()}
        ${dsFeedback()}
        ${dsOverlays()}
      </div>
    </section>

    <section class="board-section">
      <h2>Window chrome</h2>
      <div class="chrome-demo">
        <div class="chrome-panel">
          <div class="chrome-bar mac">
            <span class="traffic" aria-hidden="true"><i class="tl-close"></i><i class="tl-min"></i><i class="tl-max"></i></span>
            <div class="brand"><span class="brand-tile">B</span><span class="brand-word">Bucky</span></div>
          </div>
          <p class="chrome-caption">macOS · native frame, hidden title and traffic lights aligned to the 40 px bar.</p>
        </div>
        <div class="chrome-panel">
          <div class="chrome-bar">
            <div class="brand"><span class="brand-tile">B</span><span class="brand-word">Bucky</span></div>
            <div class="window-controls">
              <button class="window-control" type="button" aria-label="Minimize">${icon("minus")}</button>
              <button class="window-control" type="button" aria-label="Maximize">${icon("square")}</button>
              <button class="window-control close" type="button" aria-label="Close">${icon("x")}</button>
            </div>
          </div>
          <p class="chrome-caption">Windows / Linux · frameless window with renderer-drawn controls and 46 px hit targets.</p>
        </div>
      </div>
    </section>

    <section class="board-section">
      <h2>shadcn/ui mapping</h2>
      <table class="map-table">
        <thead><tr><th>Surface</th><th>Primitives</th><th>Notes</th></tr></thead>
        <tbody>
          <tr><td>Window chrome</td><td><code>custom + Window API</code></td><td>40 px drag region; native traffic lights on macOS, drawn controls elsewhere.</td></tr>
          <tr><td>Sidebar</td><td><code>ScrollArea · Button · Badge</code></td><td>Connections use the sidebar accent; read-only state carries a lock.</td></tr>
          <tr><td>Object browser</td><td><code>Table · Checkbox · DropdownMenu</code></td><td>48 px rows, sticky header, virtualized body, grid alternative.</td></tr>
          <tr><td>Breadcrumb</td><td><code>Breadcrumb</code></td><td>Bucket, prefix segments and current folder in one line.</td></tr>
          <tr><td>Search</td><td><code>Input · Command</code></td><td>Inline prefix filter; ⌘K opens a command palette later.</td></tr>
          <tr><td>Connection form</td><td><code>Dialog · Input · Select · RadioGroup · Switch</code></td><td>Two-column form, credential source and read-only guard.</td></tr>
          <tr><td>Object details</td><td><code>Sheet · Tabs · Badge · Separator</code></td><td>Right inspector, preview first, destructive action in the footer.</td></tr>
          <tr><td>Transfers</td><td><code>Sheet · Tabs · Progress</code></td><td>Active and completed groups; pause and cancel per row.</td></tr>
          <tr><td>Delete plan</td><td><code>AlertDialog · Badge</code></td><td>Plan summary, expiry and an explicit destructive action.</td></tr>
          <tr><td>Settings</td><td><code>Dialog · RadioGroup · Select</code></td><td>Vertical section nav; theme cards act as a radio group.</td></tr>
          <tr><td>Notifications</td><td><code>Sonner</code></td><td>One-line results for copy, share and completed transfers.</td></tr>
          <tr><td>Icons</td><td><code>lucide-react</code></td><td>1.6–1.8 px strokes; 14 px in controls, 16 px in empty states.</td></tr>
        </tbody>
      </table>
    </section>
  </div>`;
}

/* ------------------------------------------------------------------ */
/* Screen registry                                                     */
/* ------------------------------------------------------------------ */

const screens = [
  {
    id: "workspace",
    label: "工作区",
    title: "Workspace",
    desc: "对象浏览、批量选择、列表与网格视图；选择栏浮于内容之上。",
    tag: "Table · List / Grid",
    content: workspaceApp(),
  },
  {
    id: "welcome",
    label: "首次使用",
    title: "First run",
    desc: "零连接状态：品牌语气加单一主行动，把首次连接压缩为一步。",
    tag: "Empty state",
    content: welcomeApp(),
  },
  {
    id: "connection",
    label: "新建连接",
    title: "New connection",
    desc: "紧凑两列表单，凭证来源、只读保护与钥匙串说明在同一视野内。",
    tag: "Dialog · Form",
    content: workspaceApp({ overlay: connectionOverlay() }),
  },
  {
    id: "preview",
    label: "对象详情",
    title: "Object details",
    desc: "右侧检查器承载图片预览、对象 key、元数据与删除入口。",
    tag: "Sheet · Inspector",
    content: workspaceApp({ overlay: previewOverlay() }),
  },
  {
    id: "transfers",
    label: "传输队列",
    title: "Transfers",
    desc: "活动与已完成分组，保留速度、剩余时间、暂停与取消操作。",
    tag: "Sheet · Progress",
    content: workspaceApp({ overlay: transfersOverlay() }),
  },
  {
    id: "delete",
    label: "删除确认",
    title: "Delete plan",
    desc: "不可变计划的对象清单、总大小、有效期与不可撤销警告。",
    tag: "AlertDialog",
    content: workspaceApp({ overlay: deleteOverlay() }),
  },
  {
    id: "settings",
    label: "设置",
    title: "Settings",
    desc: "外观、网络与用量在同一对话框中分区；主题卡片可即时切换预览。",
    tag: "Dialog · Preferences",
    content: workspaceApp({ overlay: settingsOverlay() }),
  },
  {
    id: "system",
    label: "设计系统",
    title: "Design system",
    desc: "颜色、字体、圆角、间距、组件状态与 shadcn/ui 组件映射。",
    tag: "Tokens · Components",
    content: designSystemBoard(),
  },
];

function stageMarkup(screen) {
  return `<section class="stage" id="stage-${screen.id}" data-stage="${screen.id}" role="tabpanel" aria-labelledby="tab-${screen.id}">
    <div class="stage-head">
      <div><h2>${screen.title}</h2><p>${screen.desc}</p></div>
      <span class="stage-tag">${screen.tag}</span>
    </div>
    <div class="app-wrap">${screen.content}</div>
  </section>`;
}

/* ------------------------------------------------------------------ */
/* Boot                                                                */
/* ------------------------------------------------------------------ */

const params = new URLSearchParams(window.location.search);
const media = window.matchMedia("(prefers-color-scheme: dark)");
const storedTheme = window.localStorage.getItem("bucky-design-theme");
let themeMode = params.get("theme") || storedTheme || "light";
if (!["light", "dark", "system"].includes(themeMode)) themeMode = "light";

if (params.get("capture") === "1") document.body.classList.add("capture");

document.getElementById("screens").innerHTML = screens.map(stageMarkup).join("");

const bar = document.getElementById("designbar-tabs");
bar.innerHTML = screens
  .map(
    (screen) =>
      `<button type="button" role="tab" id="tab-${screen.id}" data-screen="${screen.id}" aria-selected="false" aria-controls="stage-${screen.id}">${screen.label}</button>`,
  )
  .join("");

function resolvedTheme() {
  return themeMode === "system" ? (media.matches ? "dark" : "light") : themeMode;
}

function refreshSwatchLabels() {
  const rootStyle = getComputedStyle(document.documentElement);
  for (const element of document.querySelectorAll("[data-token]")) {
    const value = rootStyle.getPropertyValue(element.dataset.token).trim();
    element.textContent = value.toUpperCase();
  }
}

function syncThemeUI() {
  for (const card of document.querySelectorAll("[data-theme-set]")) {
    card.classList.toggle("on", card.dataset.themeSet === themeMode);
  }
  const toggle = document.querySelector("[data-theme-toggle]");
  if (!toggle) return;
  const isDark = resolvedTheme() === "dark";
  toggle.innerHTML = icon(isDark ? "sun" : "moon");
  toggle.setAttribute("aria-label", isDark ? "切换到浅色主题" : "切换到深色主题");
  toggle.setAttribute("title", isDark ? "切换到浅色主题" : "切换到深色主题");
}

function applyTheme(mode, persist = true) {
  themeMode = mode;
  document.documentElement.dataset.theme = resolvedTheme();
  if (persist) window.localStorage.setItem("bucky-design-theme", mode);
  syncThemeUI();
  refreshSwatchLabels();
}

function navigate(id) {
  const target = screens.find((screen) => screen.id === id);
  if (!target) return;
  for (const stage of document.querySelectorAll(".stage")) {
    stage.classList.toggle("on", stage.dataset.stage === id);
  }
  for (const tab of document.querySelectorAll("[data-screen]")) {
    tab.setAttribute("aria-selected", String(tab.dataset.screen === id));
  }
  window.history.replaceState(null, "", `#${id}`);
  window.scrollTo({ top: 0, behavior: "instant" });
}

function updateSelection(root = document) {
  const scopes = root === document ? document.querySelectorAll(".main") : [root];
  for (const main of scopes) {
    const rowCheckboxes = main.querySelectorAll(".object-row .checkbox");
    if (!rowCheckboxes.length) continue;
    const count = main.querySelectorAll(".object-row .checkbox:checked").length;
    const barElement = main.querySelector("[data-selection-bar]");
    const countLabel = main.querySelector("[data-selection-count]");
    if (countLabel) countLabel.textContent = `${count} selected`;
    if (barElement) barElement.hidden = count === 0;
    const selectAll = main.querySelector("[data-select-all]");
    if (selectAll) {
      selectAll.checked = count > 0 && count === rowCheckboxes.length;
      selectAll.indeterminate = count > 0 && count < rowCheckboxes.length;
    }
  }
}

document.addEventListener("click", (event) => {
  const themeToggle = event.target.closest("[data-theme-toggle]");
  if (themeToggle) {
    applyTheme(resolvedTheme() === "dark" ? "light" : "dark");
    return;
  }

  const themeSet = event.target.closest("[data-theme-set]");
  if (themeSet) {
    applyTheme(themeSet.dataset.themeSet);
    return;
  }

  const viewButton = event.target.closest("[data-view]");
  if (viewButton) {
    const card = viewButton.closest(".table-card");
    const grid = viewButton.dataset.view === "grid";
    if (card) card.classList.toggle("grid-mode", grid);
    for (const button of viewButton.parentElement.querySelectorAll("[data-view]")) {
      const active = button === viewButton;
      button.classList.toggle("on", active);
      button.setAttribute("aria-pressed", String(active));
    }
    return;
  }

  const selectAll = event.target.closest("[data-select-all]");
  if (selectAll) {
    const main = selectAll.closest(".main");
    for (const rowCheckbox of main.querySelectorAll(".object-row .checkbox")) {
      rowCheckbox.checked = selectAll.checked;
      rowCheckbox.closest("tr")?.classList.toggle("selected", selectAll.checked);
    }
    updateSelection(main);
    return;
  }

  const clearSelection = event.target.closest("[data-clear-selection]");
  if (clearSelection) {
    const main = clearSelection.closest(".main");
    for (const rowCheckbox of main.querySelectorAll(".object-row .checkbox")) {
      rowCheckbox.checked = false;
      rowCheckbox.closest("tr")?.classList.remove("selected");
    }
    updateSelection(main);
    return;
  }

  const rowCheckbox = event.target.closest(".object-row .checkbox");
  if (rowCheckbox) {
    rowCheckbox.closest("tr")?.classList.toggle("selected", rowCheckbox.checked);
    updateSelection(rowCheckbox.closest(".main"));
    return;
  }

  const settingsLink = event.target.closest("[data-scroll]");
  if (settingsLink) {
    for (const button of document.querySelectorAll(".settings-nav button")) {
      button.classList.toggle("on", button === settingsLink);
    }
    document.getElementById(settingsLink.dataset.scroll)?.scrollIntoView({
      behavior: "smooth",
      block: "start",
    });
    return;
  }

  const segmentedButton = event.target.closest(".segmented button");
  if (segmentedButton) {
    for (const button of segmentedButton.parentElement.querySelectorAll("button")) {
      button.classList.toggle("on", button === segmentedButton);
    }
    return;
  }

  const switchButton = event.target.closest(".switch");
  if (switchButton) {
    switchButton.classList.toggle("on");
    switchButton.setAttribute("aria-checked", String(switchButton.classList.contains("on")));
    return;
  }

  const tabButton = event.target.closest(".tabs button");
  if (tabButton) {
    for (const button of tabButton.parentElement.querySelectorAll("button")) {
      const active = button === tabButton;
      button.classList.toggle("on", active);
      if (button.hasAttribute("aria-selected")) {
        button.setAttribute("aria-selected", String(active));
      }
    }
    return;
  }

  const screenTab = event.target.closest("[data-screen]");
  if (screenTab) {
    navigate(screenTab.dataset.screen);
    return;
  }

  const destination = event.target.closest("[data-goto]");
  if (destination) {
    if (event.target.closest("input, select, textarea")) return;
    if (
      event.target.closest(
        "button:not([data-goto]):not(.scrim):not(.sheet-scrim), .row-actions",
      )
    ) {
      return;
    }
    navigate(destination.dataset.goto);
  }
});

document.addEventListener("keydown", (event) => {
  const row = event.target.closest('tr[data-goto], .object-card[data-goto]');
  if (row && (event.key === "Enter" || event.key === " ")) {
    event.preventDefault();
    navigate(row.dataset.goto);
    return;
  }
  const tab = event.target.closest("[data-screen]");
  if (tab && (event.key === "ArrowRight" || event.key === "ArrowLeft")) {
    event.preventDefault();
    const all = [...document.querySelectorAll("[data-screen]")];
    const index = all.indexOf(tab);
    const next = all[(index + (event.key === "ArrowRight" ? 1 : -1) + all.length) % all.length];
    next.focus();
    navigate(next.dataset.screen);
  }
});

media.addEventListener("change", () => {
  if (themeMode === "system") applyTheme("system");
});

const initialScreen = params.get("screen") || window.location.hash.replace("#", "") || "workspace";
applyTheme(themeMode, false);
navigate(screens.some((screen) => screen.id === initialScreen) ? initialScreen : "workspace");
updateSelection();
refreshSwatchLabels();
