# Provider & Model Selection Fix  
`fabric-gui` – June 2025  

---

## 1. Problem Description  

The left-hand “AI Model” section in the sidebar rendered as a **blank tab**.  
Users could not:  

* Choose an AI provider (vendor)  
* Browse or select any model  
* Persist their choice in the `.env` configuration  

Result: every execution attempt failed or used hard-coded defaults.

---

## 2. Root-Cause Analysis  

| Layer | Issue | Effect |
|-------|-------|--------|
| UI layout | Provider/model widgets were created **before** vendor data loaded. | Empty controls → blank panel. |
| Data flow | Vendors and models were fetched synchronously on start-up. Any error (network, missing registry) silently aborted. | Nothing reached the widgets. |
| State mgmt | AppState did not cache models; repeated loads blocked UI. | Poor performance & flicker. |
| UX | No loading / error feedback. Collapsible section always collapsed. | User confusion. |

---

## 3. Solution Overview  

Introduce a dedicated **ModelProviderPanel** component that owns the entire provider / model experience:

1. **Async load vendors** on init → populate provider dropdown.  
2. On provider change, **async load models** only for that vendor.  
3. All UI housed inside a reusable `CollapsibleSection` so the panel can be toggled open/closed.  
4. Selections are **persisted** to `.env` via `FabricConfig`.  
5. Status label shows *loading…*, *errors*, or *model count*.

---

## 4. Implementation Details  

File | Purpose
-----|---------
`foundation/model_selection.go` | New component (`ModelProviderPanel`) – logic + UI
`foundation/layouts.go` | Sidebar now embeds `ModelProviderPanel` instead of inline code

Key points  

```go
// async vendor load
vendors, _ := app.fabricConfig.LoadVendors()
fyne.CurrentApp().Driver().RunOnMain(func() { vendorSelect.Options = vendors })

// async model load on vendor change
go mp.loadModelsForVendor(selectedVendor)

// config persistence
app.fabricConfig.SetConfig("DEFAULT_VENDOR", selected)
app.fabricConfig.SaveEnvConfig()
```

UX improvements  

* **Loading placeholders** – “Loading providers…” / “Loading models…”  
* **Disabled** model dropdown until a provider is picked.  
* **Error states** handled (`No providers available`, `Error loading models`).  
* Panel auto-expands when user first interacts.

---

## 5. Features Implemented  

* Dynamic vendor & model discovery (cached in `AppState`).  
* Collapsible “AI Model” section with toggle button.  
* Live status bar inside panel (loading, errors, model count).  
* Config-backed default selections restored on next launch.  
* Clean separation: UI ↔︎ state ↔︎ config.

---

## 6. Usage Instructions  

1. Launch Fabric-GUI.  
2. In the sidebar, click **AI Model** to expand if collapsed.  
3. Choose a *Provider* – list populates automatically.  
4. Wait a moment; *Model* dropdown fills with available models.  
5. Pick a model. Selected provider/model instantly:  
   * Stored in memory (`AppState`).  
   * Written to `.env` (`DEFAULT_VENDOR`, `DEFAULT_MODEL`).  
6. Run patterns – execution will use the chosen pair.  

---

## 7. Technical Architecture  

```
┌─────────────┐
│ FabricConfig│───► LoadVendors / LoadModelsForVendor
└────┬────────┘
     │ async (go-routine)
┌────▼────────┐
│ModelProvider│  Owns state, loading, error handling
│Panel        │
└────┬────────┘
     │ populates
┌────▼────────┐
│VendorSelect │  fyne.widget.Select
├─────────────┤
│ModelSelect  │
└─────────────┘
```

*ModelProviderPanel* updates UI on the main thread via **`fyne.CurrentApp().Driver().RunOnMain`**.  
`CollapsibleSection` wraps the panel for quick show/hide.

---

## 8. Benefits of the New Approach  

| Old | New |
|-----|-----|
| Blank panel; no feedback | Visible loading states; error messages |
| One-shot synchronous fetch | Non-blocking async loading |
| Hard to extend | Self-contained component reusable in other views |
| UI logic mixed into `layouts.go` | Clean separation (`model_selection.go`) |
| No config persistence | Provider & model saved to `.env` |

Overall the **ModelProviderPanel** delivers a robust, user-friendly, and maintainable provider/model chooser, eliminating the blank-tab issue and laying groundwork for future enhancements (model search, quotas, pricing hints, etc.).
