(() => {
    "use strict";

    // ---------------------------------------------------------------
    // Lightweight client-side column sorter for both real <table> and
    // CSS-grid pseudo-tables (.app-list-grid). Opt-in via a "sortable"
    // class on the container. A column opts out with data-nosort on
    // its header cell. Cells may declare a typed value via
    // data-sort-value (numeric/date strings preferred); otherwise the
    // visible textContent is used (with a numeric-prefix heuristic).
    //
    // Click a header triangle to open a small menu with Ascending /
    // Descending. The active column shows a filled triangle in the
    // chosen direction. Pages with server pagination get a small note
    // explaining that the sort only re-orders the current page.
    // ---------------------------------------------------------------

    const CYCLE = { none: "asc", asc: "desc", desc: "none" };

    const compareValues = (a, b) => {
        const na = parseFloat(a);
        const nb = parseFloat(b);
        const aNum = !Number.isNaN(na) && /^-?\d/.test(a.trim());
        const bNum = !Number.isNaN(nb) && /^-?\d/.test(b.trim());
        if (aNum && bNum) {
            return na - nb;
        }
        const da = Date.parse(a);
        const db = Date.parse(b);
        if (!Number.isNaN(da) && !Number.isNaN(db)) {
            return da - db;
        }
        return a.localeCompare(b, undefined, { numeric: true, sensitivity: "base" });
    };

    const cellSortValue = (cell) => {
        if (!cell) return "";
        if (cell.dataset && cell.dataset.sortValue !== undefined) {
            return cell.dataset.sortValue;
        }
        return (cell.textContent || "").trim();
    };

    let openMenu = null;
    const closeMenu = () => {
        if (openMenu && openMenu.parentNode) {
            openMenu.parentNode.removeChild(openMenu);
        }
        openMenu = null;
    };
    document.addEventListener("click", (ev) => {
        if (openMenu && !openMenu.contains(ev.target) && !ev.target.closest(".sortable-trigger")) {
            closeMenu();
        }
    });
    document.addEventListener("keydown", (ev) => {
        if (ev.key === "Escape") closeMenu();
    });

    const buildTrigger = (header, onChoose) => {
        const trigger = document.createElement("button");
        trigger.type = "button";
        trigger.className = "sortable-trigger";
        trigger.setAttribute("aria-label", "Sort column");
        trigger.innerHTML = '<span class="sortable-icon" aria-hidden="true"><span class="sortable-arrow up">\u25B2</span><span class="sortable-arrow down">\u25BC</span></span>';
        trigger.addEventListener("click", (ev) => {
            ev.stopPropagation();
            const rect = trigger.getBoundingClientRect();
            closeMenu();
            const menu = document.createElement("div");
            menu.className = "sortable-menu";
            menu.style.top = (window.scrollY + rect.bottom + 4) + "px";
            menu.style.left = (window.scrollX + rect.left) + "px";
            menu.innerHTML =
                '<button type="button" data-dir="asc">\u25B2 Sort ascending</button>' +
                '<button type="button" data-dir="desc">\u25BC Sort descending</button>' +
                '<button type="button" data-dir="none">Clear sort</button>';
            menu.addEventListener("click", (e) => {
                const btn = e.target.closest("button[data-dir]");
                if (!btn) return;
                onChoose(btn.dataset.dir);
                closeMenu();
            });
            document.body.appendChild(menu);
            openMenu = menu;
        });
        header.appendChild(trigger);
        return trigger;
    };

    const setHeaderState = (header, state) => {
        header.classList.remove("sort-asc", "sort-desc");
        if (state === "asc") header.classList.add("sort-asc");
        else if (state === "desc") header.classList.add("sort-desc");
    };

    // ---- real <table class="sortable"> ---------------------------------
    const enhanceTable = (table) => {
        const headRow = table.tHead && table.tHead.rows[0];
        const tbody = table.tBodies[0];
        if (!headRow || !tbody) return;
        const hasPagination = table.closest(".has-server-pagination");
        if (hasPagination) annotatePagination(table);
        const state = { col: -1, dir: "none" };
        const original = Array.from(tbody.rows);
        Array.from(headRow.cells).forEach((th, idx) => {
            if (th.hasAttribute("data-nosort")) return;
            th.classList.add("sortable-header");
            buildTrigger(th, (dir) => {
                if (state.col !== idx) state.dir = "none";
                state.dir = dir;
                state.col = idx;
                Array.from(headRow.cells).forEach((h) => setHeaderState(h, "none"));
                setHeaderState(th, state.dir);
                if (state.dir === "none") {
                    original.forEach((r) => tbody.appendChild(r));
                    return;
                }
                const rows = Array.from(tbody.rows);
                rows.sort((ra, rb) => {
                    const va = cellSortValue(ra.cells[idx]);
                    const vb = cellSortValue(rb.cells[idx]);
                    const cmp = compareValues(va, vb);
                    return state.dir === "asc" ? cmp : -cmp;
                });
                rows.forEach((r) => tbody.appendChild(r));
            });
        });
    };

    // ---- .app-list-grid.sortable --------------------------------------
    const enhanceGrid = (grid) => {
        const header = grid.querySelector(":scope > .app-list-grid-header");
        if (!header) return;
        const headerCells = Array.from(header.children);
        // Treat every direct child after the header that isn't an empty-state
        // helper as a data row.
        const allChildren = Array.from(grid.children);
        const rows = allChildren.filter((el) => el !== header && !el.classList.contains("app-list-empty"));
        if (rows.length === 0) return;
        const hasPagination = grid.closest(".has-server-pagination");
        if (hasPagination) annotatePagination(grid);
        const state = { col: -1, dir: "none" };
        const original = rows.slice();
        headerCells.forEach((th, idx) => {
            if (th.hasAttribute("data-nosort")) return;
            th.classList.add("sortable-header");
            buildTrigger(th, (dir) => {
                if (state.col !== idx) state.dir = "none";
                state.dir = dir;
                state.col = idx;
                headerCells.forEach((h) => setHeaderState(h, "none"));
                setHeaderState(th, state.dir);
                if (state.dir === "none") {
                    original.forEach((r) => grid.appendChild(r));
                    return;
                }
                const sorted = rows.slice().sort((ra, rb) => {
                    const va = cellSortValue(ra.children[idx]);
                    const vb = cellSortValue(rb.children[idx]);
                    const cmp = compareValues(va, vb);
                    return state.dir === "asc" ? cmp : -cmp;
                });
                sorted.forEach((r) => grid.appendChild(r));
            });
        });
    };

    const annotatePagination = (container) => {
        if (container.dataset.sortablePaginationNoted) return;
        container.dataset.sortablePaginationNoted = "1";
        const note = document.createElement("div");
        note.className = "sortable-pagination-note text-muted small";
        note.textContent = "Sort applies to the current page only.";
        container.parentNode.insertBefore(note, container);
    };

    const init = () => {
        document.querySelectorAll("table.sortable").forEach(enhanceTable);
        document.querySelectorAll(".app-list-grid.sortable").forEach(enhanceGrid);
    };

    if (document.readyState === "loading") {
        document.addEventListener("DOMContentLoaded", init);
    } else {
        init();
    }
})();
