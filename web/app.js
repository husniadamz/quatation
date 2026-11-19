const api = "/api/v1";
    const tokenKey = "quotely-session";
    const statuses = ["draft", "sent", "accepted", "rejected", "expired"];
    const money = new Intl.NumberFormat(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 });
    const state = { user: null, customers: [], templates: [], companies: [], templateSearch: "", templateStatus: "", quotePage: 1, customerPage: 1, quoteSearch: "", customerSearch: "", quoteStatus: "", previewID: null, templatePreviewURL: null, quotationPreviewLogoURL: null };
    const authForm = document.querySelector("#auth-form");
    const authFields = document.querySelector("#auth-fields");
    const authError = document.querySelector("#auth-error");
    let activeDialogQuotation = null;
    let noticeTimer;

    function showNotice(message, error = false) {
      const notice = document.querySelector("#notice");
      notice.textContent = message;
      notice.className = `notice${error ? " error" : ""}`;
      notice.hidden = false;
      clearTimeout(noticeTimer);
      noticeTimer = setTimeout(() => { notice.hidden = true; }, 4000);
    }

    async function request(path, options = {}) {
      const headers = new Headers(options.headers || {});
      const token = sessionStorage.getItem(tokenKey);
      if (token) headers.set("Authorization", `Bearer ${token}`);
      if (options.body && !(options.body instanceof FormData) && !headers.has("Content-Type")) headers.set("Content-Type", "application/json");
      const response = await fetch(`${api}${path}`, { ...options, headers });
      if (!response.ok) {
        let message = `Request failed (${response.status})`;
        try { const body = await response.json(); if (body.error) message = body.error; } catch (_) {}
        if (response.status === 401 && token) {
          sessionStorage.removeItem(tokenKey);
          showAuth(false);
        }
        throw new Error(message);
      }
      if (response.status === 204) return null;
      return response.json();
    }

    function esc(value) {
      return String(value ?? "").replace(/[&<>"']/g, (char) => ({
        "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;"
      })[char]);
    }

    function badge(status) {
      const label = status === "accepted" ? "approved" : status;
      return `<span class="status status-${esc(status)}">${esc(label)}</span>`;
    }

    function date(value) {
      if (!value) return "—";
      return new Intl.DateTimeFormat(undefined, { year: "numeric", month: "short", day: "numeric" }).format(new Date(value));
    }

    function initials(name) {
      return String(name || "Q").trim().split(/\s+/).slice(0, 2).map((part) => part[0] || "").join("").toUpperCase();
    }

    function currentRole() {
      return state.user?.role === "user" ? "sales_executive" : state.user?.role;
    }

    function roleName(role) {
      return ({
        admin: "Administrator",
        manager: "Manager",
        sales_executive: "Sales Executive",
        user: "Sales Executive"
      })[role] || "Team member";
    }

    function canManageCustomers() {
      return currentRole() === "admin" || currentRole() === "sales_executive";
    }

    function canManageTemplates() {
      return currentRole() === "admin";
    }

    function canViewTemplates() {
      return currentRole() === "admin" || currentRole() === "sales_executive";
    }

    function showAuth(setup) {
      document.querySelector("#app").hidden = true;
      document.querySelector("#auth-screen").hidden = false;
      document.querySelector("#auth-title").textContent = setup ? "Set up your workspace" : "Welcome back";
      document.querySelector("#auth-description").textContent = setup
        ? "Create your administrator account to get started. Your password must be at least 10 characters."
        : "Sign in to your quotation workspace.";
      document.querySelector("#auth-submit").textContent = setup ? "Create administrator account" : "Sign in";
      authError.textContent = "";
      authFields.replaceChildren();
      if (setup) {
        authFields.append(field("Your name", "name", "text", "Your full name", true, 150));
      }
      authFields.append(field("Email address", "email", "email", "you@company.com", true, 255));
      authFields.append(field("Password", "password", "password", setup ? "At least 10 characters" : "Your password", true, 72, setup ? 10 : 1));
      authForm.dataset.mode = setup ? "setup" : "login";
    }

    function field(labelText, name, type, placeholder, required, maxLength, minLength = 0) {
      const wrap = document.createElement("div");
      wrap.className = "field";
      const label = document.createElement("label");
      label.textContent = labelText;
      label.htmlFor = `auth-${name}`;
      const input = document.createElement("input");
      input.id = `auth-${name}`;
      input.name = name;
      input.type = type;
      input.autocomplete = name === "password" ? "current-password" : name;
      input.placeholder = placeholder;
      input.required = required;
      input.maxLength = maxLength;
      if (minLength) input.minLength = minLength;
      wrap.append(label, input);
      return wrap;
    }

    async function authenticate(payload, mode) {
      const result = await request(`/auth/${mode}`, { method: "POST", body: JSON.stringify(payload) });
      sessionStorage.setItem(tokenKey, result.data.token);
      state.user = result.data.user;
      await enterApp();
    }

    async function enterApp() {
      const result = await request("/auth/me");
      state.user = result.data;
      document.querySelector("#auth-screen").hidden = true;
      document.querySelector("#app").hidden = false;
      document.querySelector("#profile-name").textContent = state.user.name;
      const role = currentRole();
      document.querySelector("#profile-role").textContent = roleName(state.user.role);
      document.querySelector("#profile-avatar").textContent = initials(state.user.name);
      document.querySelector("#users-nav").hidden = role !== "admin";
      document.querySelector("#companies-nav").hidden = role !== "admin";
      document.querySelector("#templates-nav").hidden = !canViewTemplates();
      document.querySelector("#add-template").hidden = !canManageTemplates();
      document.querySelector("#customers-nav").hidden = !canManageCustomers();
      document.querySelector("#customer-quick-card").hidden = !canManageCustomers();
      document.querySelector("#dashboard-new-quotation").hidden = role === "manager";
      document.querySelector("#add-quotation").hidden = role === "manager";
      document.querySelector("#add-customer").hidden = !canManageCustomers();
      document.querySelector("#quotation-status-heading").textContent = role === "manager" ? "Approval" : "Status";
      document.querySelector("#today").textContent = new Intl.DateTimeFormat(undefined, { dateStyle: "medium" }).format(new Date());
      const tasks = [loadDashboard(), loadQuotations()];
      if (role === "admin") tasks.push(loadCompanies());
      if (canViewTemplates()) tasks.push(loadTemplates());
      if (canManageCustomers()) tasks.push(loadCustomers());
      await Promise.all(tasks);
    }

    async function initialize() {
      const token = sessionStorage.getItem(tokenKey);
      if (token) {
        try { await enterApp(); return; }
        catch (_) { sessionStorage.removeItem(tokenKey); }
      }
      try {
        const status = await request("/auth/setup");
        showAuth(status.data.setup_required);
      } catch (error) {
        showAuth(false);
        authError.textContent = `Unable to connect to the server: ${error.message}`;
      }
    }

    authForm.addEventListener("submit", async (event) => {
      event.preventDefault();
      const payload = Object.fromEntries(new FormData(authForm).entries());
      const mode = authForm.dataset.mode;
      authError.textContent = "";
      try { await authenticate(payload, mode); }
      catch (error) { authError.textContent = error.message; }
    });

    document.querySelector("#logout").addEventListener("click", async () => {
      try { await request("/auth/logout", { method: "POST" }); } catch (_) {}
      sessionStorage.removeItem(tokenKey);
      state.user = null;
      showAuth(false);
    });

    function navigate(page) {
      if ((page === "users" || page === "companies") && currentRole() !== "admin") return;
      if (page === "templates" && !canViewTemplates()) return;
      if (page === "customers" && !canManageCustomers()) return;
      document.querySelectorAll(".page").forEach((section) => section.classList.toggle("active", section.id === `page-${page}`));
      document.querySelectorAll(".nav [data-page]").forEach((button) => button.classList.toggle("active", button.dataset.page === page));
      document.querySelector("#topbar-title").textContent = ({ dashboard: "Dashboard", customers: "Customers", quotations: "Quotations", users: "Team members", companies: "Companies", templates: "Quotation templates" })[page];
      if (page === "dashboard") loadDashboard().catch((error) => showNotice(error.message, true));
      if (page === "customers") loadCustomers().catch((error) => showNotice(error.message, true));
      if (page === "quotations") loadQuotations().catch((error) => showNotice(error.message, true));
      if (page === "users") loadUsers().catch((error) => showNotice(error.message, true));
      if (page === "companies") loadCompanies().catch((error) => showNotice(error.message, true));
      if (page === "templates") loadTemplates().catch((error) => showNotice(error.message, true));
    }

    document.querySelectorAll("[data-page]").forEach((button) => button.addEventListener("click", () => navigate(button.dataset.page)));
    document.querySelectorAll("[data-action='new-quotation']").forEach((button) => button.addEventListener("click", () => openQuotation()));

    async function loadDashboard() {
      const result = await request("/dashboard");
      const data = result.data;
      document.querySelector("#stat-total").textContent = data.stats.total ?? 0;
      document.querySelector("#stat-pending").textContent = data.stats.pending ?? 0;
      document.querySelector("#stat-approved").textContent = data.stats.approved ?? 0;
      document.querySelector("#stat-rejected").textContent = data.stats.rejected ?? 0;
      document.querySelector("#customer-count-copy").textContent = `${data.stats.customers ?? 0} customers in your directory.`;
      const list = document.querySelector("#recent-list");
      list.replaceChildren();
      if (!data.recent_quotations.length) {
        const empty = document.createElement("div");
        empty.className = "empty";
        empty.textContent = "Your recent quotations will appear here.";
        list.append(empty);
        return;
      }
      for (const quote of data.recent_quotations) {
        const row = document.createElement("div");
        row.className = "recent-row";
        row.innerHTML = `<strong>${esc(quote.number)}</strong><span class="recent-customer">${esc(quote.customer?.name || "—")}</span>${badge(quote.status)}<strong>${money.format(quote.grand_total)}</strong>`;
        row.addEventListener("click", () => openPreview(quote.id));
        row.style.cursor = "pointer";
        list.append(row);
      }
    }

    let customerTimer;
    document.querySelector("#customer-search").addEventListener("input", (event) => {
      clearTimeout(customerTimer);
      state.customerSearch = event.target.value.trim();
      state.customerPage = 1;
      customerTimer = setTimeout(() => loadCustomers().catch((error) => showNotice(error.message, true)), 250);
    });

    async function loadCustomers() {
      const query = new URLSearchParams({ page: String(state.customerPage), page_size: "10", search: state.customerSearch });
      const result = await request(`/customers?${query}`);
      state.customers = result.data;
      const rows = document.querySelector("#customer-rows");
      rows.replaceChildren();
      for (const customer of result.data) {
        const row = document.createElement("tr");
        row.innerHTML = `<td><strong>${esc(customer.name)}</strong></td><td>${esc(customer.email || "—")}</td><td>${esc(customer.phone || "—")}</td><td>${esc(customer.address || "—")}</td>`;
        const actions = document.createElement("td");
        actions.innerHTML = `<div class="table-actions"><button class="text-button" data-edit>Edit</button><button class="text-button delete" data-delete>Delete</button></div>`;
        actions.querySelector("[data-edit]").addEventListener("click", () => openCustomer(customer));
        actions.querySelector("[data-delete]").addEventListener("click", () => deleteCustomer(customer));
        row.append(actions);
        rows.append(row);
      }
      document.querySelector("#customer-empty").hidden = result.data.length > 0;
      renderPagination("customer-pagination", result.meta, (page) => {
        state.customerPage = page;
        loadCustomers().catch((error) => showNotice(error.message, true));
      });
      const select = document.querySelector("#quote-customer");
      const selected = select.value;
      select.replaceChildren(new Option("Choose a customer", ""));
      for (const customer of state.customers) select.add(new Option(customer.name, customer.id));
      if (state.customers.some((customer) => String(customer.id) === selected)) select.value = selected;
    }

    async function loadTemplates() {
      const query = canManageTemplates() ? "" : "?active_only=true";
      const result = await request(`/templates${query}`);
      state.templates = result.data;
      const select = document.querySelector("#quote-template");
      const selected = select.value;
      select.replaceChildren(new Option("Choose a quotation template", ""));
      for (const template of state.templates.filter((item) => item.is_active)) {
        const label = `${template.company_name} — ${template.name}${template.is_default ? " (Default)" : ""}`;
        select.add(new Option(label, template.id));
      }
      if (state.templates.some((template) => String(template.id) === selected && template.is_active)) {
        select.value = selected;
      } else {
        const defaultTemplate = state.templates.find((template) => template.is_active && template.is_default) ||
          state.templates.find((template) => template.is_active);
        if (defaultTemplate) select.value = defaultTemplate.id;
      }
      if (canViewTemplates()) renderTemplates();
    }

    function renderTemplates() {
      const rows = document.querySelector("#template-rows");
      rows.replaceChildren();
      const search = state.templateSearch.toLocaleLowerCase();
      const templates = state.templates.filter((template) => {
        const matchesSearch = `${template.name} ${template.company_name}`.toLocaleLowerCase().includes(search);
        const matchesStatus = !state.templateStatus || (state.templateStatus === "active") === template.is_active;
        return matchesSearch && matchesStatus;
      });
      for (const template of templates) {
        const row = document.createElement("tr");
        row.innerHTML = `<td><div class="template-name-cell"><span class="template-file-icon" aria-hidden="true">▤</span><span><strong>${esc(template.name)}</strong><small>${template.is_default ? "Default" : ""}</small></span></div></td><td><span class="template-status ${template.is_active ? "published" : "draft"}">${template.is_active ? "Published" : "Draft"}</span></td><td>${esc(template.company_name)}</td><td>—</td><td>${date(template.created_at)}</td><td>${canManageTemplates() ? `<details class="template-actions"><summary aria-label="Actions for ${esc(template.name)}">⋮</summary><div class="template-action-menu"><button type="button" data-preview>Preview</button><button type="button" data-edit>Edit</button>${template.is_active ? "" : '<button type="button" data-toggle>Publish</button>'}${template.is_default ? "" : '<button type="button" data-default>Set as default</button>'}<button type="button" class="delete" data-delete>Delete</button></div></details>` : '<button type="button" class="text-button" data-preview>Preview</button>'}</td>`;
        const actions = row.querySelector(".template-actions");
        const menu = row.querySelector(".template-action-menu");
        actions?.addEventListener("toggle", () => {
          if (actions.open) {
            const anchor = actions.querySelector("summary").getBoundingClientRect();
            document.body.append(menu);
            menu.style.left = `${Math.max(8, Math.min(anchor.right - menu.offsetWidth, innerWidth - menu.offsetWidth - 8))}px`;
            menu.style.top = `${Math.max(8, Math.min(anchor.bottom + 5, innerHeight - menu.offsetHeight - 8))}px`;
          } else if (menu.parentElement === document.body) {
            actions.append(menu);
            menu.style.removeProperty("left");
            menu.style.removeProperty("top");
          }
        });
        row.querySelector("[data-preview]").addEventListener("click", () => { if (actions) actions.open = false; previewTemplate(template); });
        row.querySelector("[data-edit]")?.addEventListener("click", () => { actions.open = false; openTemplateEditor(template); });
        row.querySelector("[data-toggle]")?.addEventListener("click", () => { actions.open = false; setTemplateActive(template, true); });
        row.querySelector("[data-default]")?.addEventListener("click", () => { actions.open = false; setTemplateDefault(template); });
        row.querySelector("[data-delete]")?.addEventListener("click", () => { actions.open = false; deleteTemplate(template); });
        rows.append(row);
      }
      const empty = document.querySelector("#template-empty");
      empty.textContent = state.templates.length
        ? "No templates match your search or status filter."
        : "No templates yet. Create your first quotation template.";
      empty.hidden = templates.length > 0;
      document.querySelector(".template-table").hidden = templates.length === 0;
    }

    document.querySelector("#template-search").addEventListener("input", (event) => {
      state.templateSearch = event.target.value.trim();
      renderTemplates();
    });
    document.querySelector("#template-status-filter").addEventListener("change", (event) => {
      state.templateStatus = event.target.value;
      renderTemplates();
    });

    async function setTemplateActive(template, active) {
      try {
        await request(`/templates/${template.id}/active`, { method: "PATCH", body: JSON.stringify({ is_active: active }) });
        await loadTemplates();
        showNotice(active ? "Template activated." : "Template deactivated.");
      } catch (error) { showNotice(error.message, true); }
    }

    async function setTemplateDefault(template) {
      try {
        await request(`/templates/${template.id}/default`, { method: "PATCH", body: "{}" });
        await loadTemplates();
        showNotice("Default template updated for this company.");
      } catch (error) { showNotice(error.message, true); }
    }

    async function deleteTemplate(template) {
      if (!confirm(`Delete template "${template.name}"? Existing quotation PDFs keep their saved template snapshot.`)) return;
      try {
        await request(`/templates/${template.id}`, { method: "DELETE" });
        await loadTemplates();
        showNotice("Template deleted.");
      } catch (error) { showNotice(error.message, true); }
    }

    document.querySelector("#add-template").addEventListener("click", () => openTemplateEditor());

    async function openTemplateEditor(template = null, previewOnly = false) {
      const form = document.querySelector("#template-form");
      form.reset();
      form.dataset.id = template?.id || "";
      const companySelect = document.querySelector("#template-company");
      companySelect.replaceChildren(new Option("Choose a company", ""));
      for (const company of state.companies) companySelect.add(new Option(company.name, company.id));
      if (template && !state.companies.some((company) => company.id === template.company_id)) {
        companySelect.add(new Option(template.company_name, template.company_id));
      }
      companySelect.value = template?.company_id || "";
      companySelect.disabled = !canManageTemplates();
      document.querySelector("#template-dialog-title").textContent = template ? "Edit quotation template" : "New quotation template";
      form.querySelector('[type="submit"]').textContent = template ? "Save changes" : "Save template";
      document.querySelector("#template-active").checked = template?.is_active ?? true;
      document.querySelector("#template-default").checked = template?.is_default ?? false;
      document.querySelector("#template-primary").value = template?.settings.primary_color || "#264a73";
      document.querySelector("#template-accent").value = template?.settings.accent_color || "#d6a94b";
      document.querySelector("#template-title").value = template?.settings.header_title || "Quotation";
      document.querySelector("#template-font").value = template?.settings.font_family || "Arial";
      document.querySelector("#template-layout").value = template?.settings.layout || "portrait";
      for (const [field, key] of [
        ["template-name", "name"],
        ["template-email", "company_email"], ["template-phone", "company_phone"],
        ["template-address", "company_address"], ["template-header", "header_text"],
        ["template-footer", "footer_text"], ["template-payment", "payment_details"],
        ["template-terms", "terms"], ["template-signature", "signature_label"]
      ]) {
        const setting = template?.settings?.[key];
        document.querySelector(`#${field}`).value = field === "template-name" ? template?.[key] || "" : setting || "";
      }
      document.querySelector("#template-logo").value = "";
      if (state.templateLogoURL) URL.revokeObjectURL(state.templateLogoURL);
      state.templateLogoURL = "";
      updateTemplatePreview();
      document.querySelector("#template-dialog").showModal();
      if (template?.logo_url) {
        try {
          const response = await fetch(template.logo_url, {
            headers: { Authorization: `Bearer ${sessionStorage.getItem(tokenKey)}` }
          });
          if (!response.ok) throw new Error("Unable to load template logo");
          state.templateLogoURL = URL.createObjectURL(await response.blob());
          updateTemplatePreview();
        } catch (error) {
          showNotice(error.message, true);
        }
      }
      if (previewOnly) {
        document.querySelector("#template-dialog-title").textContent = `${template.name} · Preview`;
        form.querySelector('[type="submit"]').hidden = true;
        const headerClose = form.querySelector(".dialog-head [data-close='template-dialog']");
        headerClose.setAttribute("aria-label", "Close preview");
        headerClose.title = "Close preview";
        form.querySelector(".dialog-foot [data-close='template-dialog']").textContent = "Close preview";
        form.querySelectorAll("input,select,textarea").forEach((input) => { input.disabled = true; });
      }
    }

    function templateFormValues() {
      const companySelect = document.querySelector("#template-company");
      return {
        company_id: Number(companySelect.value),
        company_name: companySelect.selectedOptions[0]?.textContent || "",
        name: document.querySelector("#template-name").value,
        settings: {
          company_email: document.querySelector("#template-email").value,
          company_phone: document.querySelector("#template-phone").value,
          company_address: document.querySelector("#template-address").value,
          header_title: document.querySelector("#template-title").value,
          header_text: document.querySelector("#template-header").value,
          footer_text: document.querySelector("#template-footer").value,
          primary_color: document.querySelector("#template-primary").value,
          accent_color: document.querySelector("#template-accent").value,
          font_family: document.querySelector("#template-font").value,
          layout: document.querySelector("#template-layout").value,
          terms: document.querySelector("#template-terms").value,
          payment_details: document.querySelector("#template-payment").value,
          signature_label: document.querySelector("#template-signature").value
        },
        is_active: document.querySelector("#template-active").checked,
        is_default: document.querySelector("#template-default").checked
      };
    }

    function updateTemplatePreview(logoURL = state.templateLogoURL || "") {
      const values = templateFormValues();
      const settings = values.settings;
      const preview = document.querySelector("#template-live-preview");
      preview.className = `template-paper ${settings.layout}`;
      preview.style.setProperty("--paper-primary", settings.primary_color);
      preview.style.setProperty("--paper-accent", settings.accent_color);
      preview.style.setProperty("--paper-font", settings.font_family);
      document.querySelector("#template-preview-layout").textContent = `A4 · ${settings.layout}`;
      preview.innerHTML = `<div class="paper-header">${logoURL ? `<img class="paper-logo" src="${esc(logoURL)}" alt="Company logo">` : '<span class="paper-logo-placeholder">LOGO</span>'}<div class="paper-brand"><h3>${esc(values.company_name || "Your company")}</h3><span>${esc(settings.company_address || "Company address")}</span><span>${esc(settings.company_email || "email@company.com")}${settings.company_phone ? ` · ${esc(settings.company_phone)}` : ""}</span></div></div><div class="paper-title"><div><h2>${esc(settings.header_title || "Quotation")}</h2><p>${esc(settings.header_text || "Prepared for your consideration")}</p></div><span>QUOTE-0001<br>Issued today</span></div><div class="paper-customer"><small>PREPARED FOR</small><strong>Customer name</strong><span>customer@example.com</span></div><div class="paper-table"><div class="paper-table-head"><span>Description</span><span>Qty</span><span>Unit price</span><span>Total</span></div><div><span>Professional services</span><span>2</span><span>125.00</span><span>250.00</span></div><div><span>Setup and delivery</span><span>1</span><span>80.00</span><span>80.00</span></div></div><div class="paper-totals"><span>Subtotal</span><strong>330.00</strong><span>Tax</span><strong>0.00</strong><b>Grand total</b><b>330.00</b></div>${settings.payment_details ? `<div class="paper-section"><b>Payment details</b><span>${esc(settings.payment_details)}</span></div>` : ""}${settings.terms ? `<div class="paper-section"><b>Terms and conditions</b><span>${esc(settings.terms)}</span></div>` : ""}${settings.signature_label ? `<div class="paper-signature">${esc(settings.signature_label)}<i></i></div>` : ""}<div class="paper-footer">${esc(settings.footer_text || "Thank you for your business.")}</div>`;
    }

    for (const selector of ["#template-company", "#template-name", "#template-email", "#template-phone", "#template-address", "#template-title", "#template-header", "#template-footer", "#template-payment", "#template-terms", "#template-signature", "#template-primary", "#template-accent", "#template-font", "#template-layout"]) {
      document.querySelector(selector).addEventListener("input", () => updateTemplatePreview());
      document.querySelector(selector).addEventListener("change", () => updateTemplatePreview());
    }

    document.querySelector("#template-logo").addEventListener("change", (event) => {
      const file = event.target.files[0];
      if (!file) return;
      if (!["image/png", "image/jpeg"].includes(file.type) || file.size > 2 * 1024 * 1024) {
        event.target.value = "";
        showNotice("Choose a PNG or JPEG logo no larger than 2 MB.", true);
        return;
      }
      state.templateLogoURL = URL.createObjectURL(file);
      updateTemplatePreview();
    });

    document.querySelector("#template-form").addEventListener("submit", async (event) => {
      event.preventDefault();
      const form = event.currentTarget;
      const id = form.dataset.id;
      const file = document.querySelector("#template-logo").files[0];
      const payload = templateFormValues();
      if (!payload.company_id) {
        showNotice("Choose a registered company for this template.", true);
        return;
      }
      if (payload.is_default && !payload.is_active) {
        showNotice("A default template must be active.", true);
        return;
      }
      try {
        const result = await request(id ? `/templates/${id}` : "/templates", {
          method: id ? "PUT" : "POST", body: JSON.stringify(payload)
        });
        const templateID = id || result.data.id;
        if (file) {
          const upload = new FormData();
          upload.append("logo", file);
          await request(`/templates/${templateID}/logo`, { method: "POST", body: upload });
        }
        document.querySelector("#template-dialog").close();
        if (state.templateLogoURL) URL.revokeObjectURL(state.templateLogoURL);
        state.templateLogoURL = null;
        await loadTemplates();
        showNotice(id ? "Template updated." : "Template created.");
      } catch (error) { showNotice(error.message, true); }
    });

    async function previewTemplate(template) {
      await openTemplateEditor(template, true);
    }

    function resetTemplateEditor() {
      const form = document.querySelector("#template-form");
      form.querySelectorAll("input,select,textarea").forEach((input) => { input.disabled = false; });
      form.querySelector('[type="submit"]').hidden = false;
      const headerClose = form.querySelector(".dialog-head [data-close='template-dialog']");
      headerClose.setAttribute("aria-label", "Close");
      headerClose.removeAttribute("title");
      form.querySelector(".dialog-foot [data-close='template-dialog']").textContent = "Cancel";
      if (state.templateLogoURL) URL.revokeObjectURL(state.templateLogoURL);
      state.templateLogoURL = null;
    }

    document.querySelector("#template-dialog").addEventListener("close", resetTemplateEditor);

    async function deleteCustomer(customer) {
      if (!confirm(`Delete ${customer.name}?`)) return;
      try {
        await request(`/customers/${customer.id}`, { method: "DELETE" });
        await Promise.all([loadCustomers(), loadDashboard()]);
        showNotice("Customer deleted.");
      } catch (error) { showNotice(error.message, true); }
    }

    function renderPagination(id, meta, onPage) {
      const root = document.querySelector(`#${id}`);
      root.replaceChildren();
      const pages = Math.max(1, Math.ceil(meta.total / meta.page_size));
      const summary = document.createElement("span");
      const start = meta.total ? (meta.page - 1) * meta.page_size + 1 : 0;
      const end = Math.min(meta.page * meta.page_size, meta.total);
      summary.textContent = `${start}–${end} of ${meta.total}`;
      const controls = document.createElement("div");
      controls.className = "pagination-controls";
      for (const [label, target, disabled] of [["← Previous", meta.page - 1, meta.page <= 1], ["Next →", meta.page + 1, meta.page >= pages]]) {
        const button = document.createElement("button");
        button.className = "button secondary";
        button.textContent = label;
        button.disabled = disabled;
        button.addEventListener("click", () => onPage(target));
        controls.append(button);
      }
      root.append(summary, controls);
    }

    document.querySelector("#add-customer").addEventListener("click", () => openCustomer());
    function openCustomer(customer = null) {
      const form = document.querySelector("#customer-form");
      form.reset();
      form.dataset.id = customer?.id || "";
      document.querySelector("#customer-dialog-title").textContent = customer ? "Edit customer" : "Add customer";
      if (customer) for (const key of ["name", "email", "phone", "address"]) form.elements[key].value = customer[key] || "";
      document.querySelector("#customer-dialog").showModal();
    }

    document.querySelector("#customer-form").addEventListener("submit", async (event) => {
      event.preventDefault();
      const form = event.currentTarget;
      const id = form.dataset.id;
      const payload = Object.fromEntries(new FormData(form).entries());
      try {
        await request(id ? `/customers/${id}` : "/customers", { method: id ? "PUT" : "POST", body: JSON.stringify(payload) });
        document.querySelector("#customer-dialog").close();
        await Promise.all([loadCustomers(), loadDashboard()]);
        showNotice(id ? "Customer updated." : "Customer created.");
      } catch (error) { showNotice(error.message, true); }
    });

    let quoteTimer;
    document.querySelector("#quotation-search").addEventListener("input", (event) => {
      clearTimeout(quoteTimer);
      state.quoteSearch = event.target.value.trim();
      state.quotePage = 1;
      quoteTimer = setTimeout(() => loadQuotations().catch((error) => showNotice(error.message, true)), 250);
    });
    document.querySelector("#quotation-filter").addEventListener("change", (event) => {
      state.quoteStatus = event.target.value;
      state.quotePage = 1;
      loadQuotations().catch((error) => showNotice(error.message, true));
    });

    async function loadQuotations() {
      const query = new URLSearchParams({ page: String(state.quotePage), page_size: "10", search: state.quoteSearch });
      if (state.quoteStatus) query.set("status", state.quoteStatus);
      const result = await request(`/quotations?${query}`);
      const rows = document.querySelector("#quotation-rows");
      rows.replaceChildren();
      for (const quote of result.data) {
        const row = document.createElement("tr");
        const issueDate = date(quote.created_at);
        row.innerHTML = `<td><strong>${esc(quote.number)}</strong></td><td>${esc(quote.customer?.name || "—")}</td><td>${issueDate}</td>`;
        const statusCell = document.createElement("td");
        if (currentRole() === "admin") {
          const select = document.createElement("select");
          select.className = "table-status";
          select.setAttribute("aria-label", `Status for ${quote.number}`);
          statuses.forEach((value) => select.add(new Option(value === "accepted" ? "approved" : value, value)));
          select.value = quote.status;
          select.addEventListener("change", async () => {
            const previous = quote.status;
            try {
              await request(`/quotations/${quote.id}/status`, { method: "PATCH", body: JSON.stringify({ status: select.value }) });
              quote.status = select.value;
              await Promise.all([loadDashboard(), loadQuotations()]);
              showNotice("Quotation status updated.");
            } catch (error) { select.value = previous; showNotice(error.message, true); }
          });
          statusCell.append(select);
        } else {
          statusCell.innerHTML = badge(quote.status);
        }
        row.append(statusCell);
        const total = document.createElement("td");
        total.innerHTML = `<strong>${money.format(quote.grand_total)}</strong>`;
        row.append(total);
        const actions = document.createElement("td");
        if (currentRole() === "admin") {
          actions.innerHTML = `<div class="table-actions"><button class="text-button" data-preview>View</button><button class="text-button" data-edit>Edit</button><button class="text-button delete" data-delete>Delete</button></div>`;
        } else if (currentRole() === "sales_executive") {
          actions.innerHTML = `<div class="table-actions"><button class="text-button" data-preview>View</button><button class="text-button" data-edit>Edit</button></div>`;
        } else {
          actions.innerHTML = `<div class="table-actions"><button class="text-button" data-preview>Review</button>${["draft", "sent"].includes(quote.status) ? '<button class="text-button approve" data-approve>Approve</button><button class="text-button delete" data-reject>Reject</button>' : ""}</div>`;
        }
        actions.querySelector("[data-preview]").addEventListener("click", () => openPreview(quote.id));
        actions.querySelector("[data-edit]")?.addEventListener("click", () => editQuotation(quote.id));
        actions.querySelector("[data-delete]")?.addEventListener("click", () => deleteQuotation(quote));
        actions.querySelector("[data-approve]")?.addEventListener("click", () => updateApproval(quote, "accepted"));
        actions.querySelector("[data-reject]")?.addEventListener("click", () => updateApproval(quote, "rejected"));
        row.append(actions);
        rows.append(row);
      }
      document.querySelector("#quotation-empty").hidden = result.data.length > 0;
      renderPagination("quotation-pagination", result.meta, (page) => {
        state.quotePage = page;
        loadQuotations().catch((error) => showNotice(error.message, true));
      });
    }

    document.querySelector("#add-quotation").addEventListener("click", () => openQuotation());
    function openQuotation(quotation = null) {
      if (!canManageCustomers()) return;
      if (!state.customers.length) {
        navigate("customers");
        showNotice("Add a customer before creating a quotation.", true);
        return;
      }
      activeDialogQuotation = quotation;
      const form = document.querySelector("#quotation-form");
      form.reset();
      document.querySelector("#quotation-dialog-title").textContent = quotation ? "Edit quotation" : "New quotation";
      form.querySelector('[type="submit"]').textContent = quotation ? "Save changes" : "Create quotation";
      const select = document.querySelector("#quote-customer");
      select.replaceChildren(new Option("Choose a customer", ""));
      for (const customer of state.customers) select.add(new Option(customer.name, customer.id));
      if (quotation) {
        select.value = quotation.customer_id;
        if (quotation.template_id && state.templates.some((template) => template.id === quotation.template_id && template.is_active)) {
          document.querySelector("#quote-template").value = quotation.template_id;
        }
        document.querySelector("#quote-tax").value = quotation.tax_rate;
        document.querySelector("#quote-notes").value = quotation.notes || "";
        document.querySelector("#quote-valid").value = quotation.valid_until ? quotation.valid_until.slice(0, 10) : "";
      }
      const container = document.querySelector("#quote-items");
      container.replaceChildren();
      const items = quotation?.items?.length ? quotation.items : [{ description: "", quantity: "", unit_price: "" }];
      items.forEach((item) => addItemRow(item.description, item.quantity, item.unit_price));
      updateQuoteEstimate();
      document.querySelector("#quotation-dialog").showModal();
    }

    async function editQuotation(id) {
      try {
        const result = await request(`/quotations/${id}`);
        openQuotation(result.data);
      } catch (error) { showNotice(error.message, true); }
    }

    async function deleteQuotation(quote) {
      if (!confirm(`Delete quotation ${quote.number}?`)) return;
      try {
        await request(`/quotations/${quote.id}`, { method: "DELETE" });
        await Promise.all([loadQuotations(), loadDashboard()]);
        showNotice("Quotation deleted.");
      } catch (error) { showNotice(error.message, true); }
    }

    async function updateApproval(quote, status) {
      try {
        await request(`/quotations/${quote.id}/status`, { method: "PATCH", body: JSON.stringify({ status }) });
        await Promise.all([loadQuotations(), loadDashboard()]);
        showNotice(status === "accepted" ? "Quotation approved." : "Quotation rejected.");
        return true;
      } catch (error) { showNotice(error.message, true); }
      return false;
    }

    function addItemRow(description = "", quantity = "", unitPrice = "") {
      const row = document.createElement("div");
      row.className = "item-editor";
      const desc = document.createElement("input");
      desc.placeholder = "Item description";
      desc.setAttribute("aria-label", "Item description");
      desc.maxLength = 255;
      desc.required = true;
      desc.value = description;
      const qty = document.createElement("input");
      qty.type = "number"; qty.min = "0.01"; qty.step = "0.01"; qty.placeholder = "1";
      qty.setAttribute("aria-label", "Quantity"); qty.required = true; qty.value = quantity;
      const price = document.createElement("input");
      price.type = "number"; price.min = "0"; price.step = "0.01"; price.placeholder = "0.00";
      price.setAttribute("aria-label", "Unit price"); price.required = true; price.value = unitPrice;
      const remove = document.createElement("button");
      remove.type = "button"; remove.textContent = "×"; remove.setAttribute("aria-label", "Remove item");
      remove.addEventListener("click", () => { if (document.querySelectorAll(".item-editor").length > 1) row.remove(); updateQuoteEstimate(); });
      qty.addEventListener("input", updateQuoteEstimate);
      price.addEventListener("input", updateQuoteEstimate);
      row.append(desc, qty, price, remove);
      document.querySelector("#quote-items").append(row);
    }

    function updateQuoteEstimate() {
      let subtotal = 0;
      for (const row of document.querySelectorAll(".item-editor")) {
        const inputs = row.querySelectorAll("input");
        subtotal += Math.round((Number(inputs[1].value || 0) * Number(inputs[2].value || 0) + Number.EPSILON) * 100) / 100;
      }
      const taxRate = Number(document.querySelector("#quote-tax").value || 0);
      const tax = Math.round((subtotal * taxRate / 100 + Number.EPSILON) * 100) / 100;
      document.querySelector("#quote-estimate").textContent = `Total: ${money.format(subtotal + tax)}`;
      document.querySelector("#quote-totals").innerHTML = `<div class="total-row"><span>Subtotal</span><span>${money.format(subtotal)}</span></div><div class="total-row"><span>Tax (${money.format(taxRate)}%)</span><span>${money.format(tax)}</span></div><div class="total-row final"><span>Grand total</span><span>${money.format(subtotal + tax)}</span></div>`;
    }

    document.querySelector("#quote-tax").addEventListener("input", updateQuoteEstimate);
    document.querySelector("#add-item").addEventListener("click", () => addItemRow());
    document.querySelector("#quotation-form").addEventListener("submit", async (event) => {
      event.preventDefault();
      const items = [...document.querySelectorAll(".item-editor")].map((row) => {
        const inputs = row.querySelectorAll("input");
        return { description: inputs[0].value, quantity: Number(inputs[1].value), unit_price: Number(inputs[2].value) };
      });
      const valid = document.querySelector("#quote-valid").value;
      const payload = {
        customer_id: Number(document.querySelector("#quote-customer").value),
        template_id: Number(document.querySelector("#quote-template").value),
        tax_rate: Number(document.querySelector("#quote-tax").value),
        notes: document.querySelector("#quote-notes").value,
        valid_until: valid ? `${valid}T00:00:00Z` : null,
        items
      };
      const id = activeDialogQuotation?.id;
      try {
        await request(id ? `/quotations/${id}` : "/quotations", { method: id ? "PUT" : "POST", body: JSON.stringify(payload) });
        document.querySelector("#quotation-dialog").close();
        await Promise.all([loadQuotations(), loadDashboard()]);
        showNotice(id ? "Quotation updated." : "Quotation created.");
      } catch (error) { showNotice(error.message, true); }
    });

    async function openPreview(id) {
      try {
        const result = await request(`/quotations/${id}`);
        const quote = result.data;
        state.previewID = id;
        const snapshot = quote.template_snapshot || {};
        const templateSettings = snapshot.settings || {};
        if (state.quotationPreviewLogoURL) URL.revokeObjectURL(state.quotationPreviewLogoURL);
        state.quotationPreviewLogoURL = null;
        if (snapshot.logo_url) {
          const logoResponse = await fetch(snapshot.logo_url, {
            headers: { Authorization: `Bearer ${sessionStorage.getItem(tokenKey)}` }
          });
          if (!logoResponse.ok) throw new Error("Unable to load the quotation template logo");
          state.quotationPreviewLogoURL = URL.createObjectURL(await logoResponse.blob());
        }
        const items = quote.items.map((item) => `<tr><td>${esc(item.description)}</td><td style="text-align:right">${money.format(item.quantity)}</td><td style="text-align:right">${money.format(item.unit_price)}</td><td style="text-align:right">${money.format(item.line_total)}</td></tr>`).join("");
        document.querySelector("#preview-content").innerHTML = `<article class="preview">
          ${state.quotationPreviewLogoURL ? `<img class="quotation-preview-logo" src="${esc(state.quotationPreviewLogoURL)}" alt="Company logo">` : ""}
          <div class="preview-brand" style="--preview-primary:${esc(templateSettings.primary_color || "#3c5fc9")};--preview-accent:${esc(templateSettings.accent_color || "#d6a94b")};font-family:${esc(templateSettings.font_family || "Arial")}"><div><h2>${esc(templateSettings.header_title || "QUOTATION")}</h2><p>${esc(snapshot.company_name || "Prepared for your consideration")}</p>${templateSettings.company_address ? `<p>${esc(templateSettings.company_address)}</p>` : ""}${templateSettings.company_email || templateSettings.company_phone ? `<p>${esc([templateSettings.company_email, templateSettings.company_phone].filter(Boolean).join(" · "))}</p>` : ""}${templateSettings.header_text ? `<p>${esc(templateSettings.header_text)}</p>` : ""}</div><div class="preview-number"><strong>${esc(quote.number)}</strong><br><br>Issued ${date(quote.created_at)}<br>Valid until ${date(quote.valid_until)}</div></div>
          <div class="preview-customer"><small>Prepared for</small><strong>${esc(quote.customer?.name || "")}</strong><p>${esc(quote.customer?.email || "")}${quote.customer?.phone ? ` · ${esc(quote.customer.phone)}` : ""}${quote.customer?.address ? `\n${esc(quote.customer.address)}` : ""}</p></div>
          <div class="table-wrap preview-table"><table><thead><tr><th>Description</th><th style="text-align:right">Qty</th><th style="text-align:right">Unit price</th><th style="text-align:right">Total</th></tr></thead><tbody>${items}</tbody></table></div>
          <div class="totals preview-totals"><div class="total-row"><span>Subtotal</span><span>${money.format(quote.subtotal)}</span></div><div class="total-row"><span>Tax (${money.format(quote.tax_rate)}%)</span><span>${money.format(quote.tax_amount)}</span></div><div class="total-row final"><span>Grand total</span><span>${money.format(quote.grand_total)}</span></div></div>
          ${quote.notes ? `<div class="preview-notes"><strong>Notes</strong><br><br>${esc(quote.notes)}</div>` : ""}
          ${templateSettings.payment_details ? `<div class="preview-notes"><strong>Payment details</strong><br><br>${esc(templateSettings.payment_details)}</div>` : ""}
          ${templateSettings.terms ? `<div class="preview-notes"><strong>Terms and conditions</strong><br><br>${esc(templateSettings.terms)}</div>` : ""}
          ${templateSettings.signature_label ? `<div class="preview-signature">${esc(templateSettings.signature_label)}</div>` : ""}
          ${templateSettings.footer_text ? `<div class="quotation-preview-footer">${esc(templateSettings.footer_text)}</div>` : ""}
        </article>`;
        const approvalActions = document.querySelector("#preview-approval-actions");
        approvalActions.replaceChildren();
        approvalActions.hidden = currentRole() !== "manager" || !["draft", "sent"].includes(quote.status);
        if (!approvalActions.hidden) {
          approvalActions.className = "approval-actions";
          for (const [label, status] of [["Approve", "accepted"], ["Reject", "rejected"]]) {
            const button = document.createElement("button");
            button.className = status === "accepted" ? "button small" : "button danger small";
            button.type = "button";
            button.textContent = label;
            button.addEventListener("click", async () => {
              if (await updateApproval(quote, status)) document.querySelector("#preview-dialog").close();
            });
            approvalActions.append(button);
          }
        }
        document.querySelector("#preview-dialog").showModal();
      } catch (error) { showNotice(error.message, true); }
    }

    document.querySelector("#preview-print").addEventListener("click", () => window.print());
    document.querySelector("#preview-dialog").addEventListener("close", () => {
      if (state.quotationPreviewLogoURL) URL.revokeObjectURL(state.quotationPreviewLogoURL);
      state.quotationPreviewLogoURL = null;
    });
    document.querySelector("#preview-pdf").addEventListener("click", async () => {
      try {
        const response = await fetch(`${api}/quotations/${state.previewID}/pdf`, { headers: { Authorization: `Bearer ${sessionStorage.getItem(tokenKey)}` } });
        if (!response.ok) throw new Error("Unable to download quotation PDF");
        const blob = await response.blob();
        const url = URL.createObjectURL(blob);
        const link = document.createElement("a");
        link.href = url; link.download = "quotation.pdf"; link.click();
        URL.revokeObjectURL(url);
      } catch (error) { showNotice(error.message, true); }
    });

    document.querySelectorAll("[data-close]").forEach((button) => button.addEventListener("click", () => document.querySelector(`#${button.dataset.close}`).close()));
    for (const dialog of document.querySelectorAll("dialog")) dialog.addEventListener("click", (event) => {
      if (event.target === dialog) dialog.close();
    });

    document.querySelector("#add-user").addEventListener("click", () => {
      if (!state.companies.length) {
        showNotice("Register a company before inviting a team member.", true);
        navigate("companies");
        return;
      }
      openUserEditor();
    });
    document.querySelector("#user-form").addEventListener("submit", async (event) => {
      event.preventDefault();
      const form = event.currentTarget;
      const id = form.dataset.id;
      const payload = Object.fromEntries(new FormData(form).entries());
      payload.company_id = Number(payload.company_id);
      if (id) delete payload.password;
      try {
        await request(id ? `/users/${id}` : "/users", {
          method: id ? "PUT" : "POST", body: JSON.stringify(payload)
        });
        document.querySelector("#user-dialog").close();
        await loadUsers();
        showNotice(id ? "Team member updated." : "Team member created.");
      } catch (error) { showNotice(error.message, true); }
    });

    function openUserEditor(user = null) {
      const form = document.querySelector("#user-form");
      form.reset();
      form.dataset.id = user?.id || "";
      document.querySelector("#user-dialog-title").textContent = user ? "Edit team member" : "Invite team member";
      document.querySelector("#user-submit").textContent = user ? "Save changes" : "Create account";
      document.querySelector("#user-password-field").hidden = Boolean(user);
      document.querySelector("#user-password").required = !user;
      const companySelect = document.querySelector("#user-company");
      companySelect.replaceChildren(new Option("Choose a company", ""));
      for (const company of state.companies) companySelect.add(new Option(company.name, company.id));
      if (user) {
        document.querySelector("#user-name").value = user.name;
        document.querySelector("#user-email").value = user.email;
        document.querySelector("#user-company").value = user.company_id || "";
        document.querySelector("#user-role").value = user.role === "user" ? "sales_executive" : user.role;
      }
      document.querySelector("#user-dialog").showModal();
    }

    async function loadUsers() {
      if (currentRole() !== "admin") return;
      const result = await request("/users");
      const rows = document.querySelector("#user-rows");
      rows.replaceChildren();
      for (const user of result.data) {
        const row = document.createElement("tr");
        row.innerHTML = `<td><strong>${esc(user.name)}</strong></td><td>${esc(user.email)}</td><td>${esc(user.company?.name || "—")}</td><td>${esc(roleName(user.role))}</td><td>${date(user.created_at)}</td>`;
        const action = document.createElement("td");
        if (user.id === state.user.id) action.textContent = "You";
        else {
          const edit = document.createElement("button");
          edit.className = "text-button";
          edit.textContent = "Edit";
          edit.addEventListener("click", () => openUserEditor(user));
          const remove = document.createElement("button");
          remove.className = "text-button delete"; remove.textContent = "Remove";
          remove.addEventListener("click", async () => {
            if (!confirm(`Remove ${user.name} from the workspace?`)) return;
            try { await request(`/users/${user.id}`, { method: "DELETE" }); await loadUsers(); showNotice("Team member removed."); }
            catch (error) { showNotice(error.message, true); }
          });
          action.append(edit, remove);
        }
        row.append(action);
        rows.append(row);
      }
      document.querySelector("#user-empty").hidden = result.data.length > 0;
    }

    document.querySelector("#add-company").addEventListener("click", () => openCompanyEditor());
    document.querySelector("#company-form").addEventListener("submit", async (event) => {
      event.preventDefault();
      const form = event.currentTarget;
      const id = form.dataset.id;
      const payload = Object.fromEntries(new FormData(form).entries());
      try {
        await request(id ? `/companies/${id}` : "/companies", {
          method: id ? "PUT" : "POST", body: JSON.stringify(payload)
        });
        document.querySelector("#company-dialog").close();
        await loadCompanies();
        showNotice(id ? "Company updated." : "Company registered.");
      } catch (error) { showNotice(error.message, true); }
    });

    function openCompanyEditor(company = null) {
      const form = document.querySelector("#company-form");
      form.reset();
      form.dataset.id = company?.id || "";
      document.querySelector("#company-dialog-title").textContent = company ? "Update company" : "Register company";
      for (const key of ["name", "email", "phone", "address"]) {
        document.querySelector(`#company-${key}`).value = company?.[key] || "";
      }
      document.querySelector("#company-dialog").showModal();
    }

    async function loadCompanies() {
      if (currentRole() !== "admin") return;
      const result = await request("/companies");
      state.companies = result.data;
      const rows = document.querySelector("#company-rows");
      rows.replaceChildren();
      for (const company of state.companies) {
        const row = document.createElement("tr");
        row.innerHTML = `<td><strong>${esc(company.name)}</strong></td><td>${esc(company.email || "—")}</td><td>${esc(company.phone || "—")}</td><td>${esc(company.address || "—")}</td>`;
        const actions = document.createElement("td");
        const members = document.createElement("button");
        members.className = "text-button";
        members.textContent = "Members";
        members.addEventListener("click", () => openCompanyMembers(company));
        const edit = document.createElement("button");
        edit.className = "text-button";
        edit.textContent = "Edit";
        edit.addEventListener("click", () => openCompanyEditor(company));
        const remove = document.createElement("button");
        remove.className = "text-button delete";
        remove.textContent = "Delete";
        remove.addEventListener("click", async () => {
          if (!confirm(`Delete ${company.name}? Companies assigned to team members cannot be deleted.`)) return;
          try {
            await request(`/companies/${company.id}`, { method: "DELETE" });
            await loadCompanies();
            showNotice("Company deleted.");
          } catch (error) { showNotice(error.message, true); }
        });
        actions.append(members, edit, remove);
        row.append(actions);
        rows.append(row);
      }
      document.querySelector("#company-empty").hidden = state.companies.length > 0;
    }

    async function openCompanyMembers(company) {
      try {
        const result = await request(`/companies/${company.id}/users`);
        const rows = document.querySelector("#company-member-rows");
        rows.replaceChildren();
        for (const user of result.data) {
          const row = document.createElement("tr");
          row.innerHTML = `<td><strong>${esc(user.name)}</strong></td><td>${esc(user.email)}</td><td>${esc(roleName(user.role))}</td><td>${date(user.created_at)}</td>`;
          rows.append(row);
        }
        document.querySelector("#company-members-title").textContent = `${company.name} · Members`;
        document.querySelector("#company-members-empty").hidden = result.data.length > 0;
        document.querySelector("#company-members-dialog").showModal();
      } catch (error) { showNotice(error.message, true); }
    }

    initialize().catch((error) => { showAuth(false); authError.textContent = error.message; });
