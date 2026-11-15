# Quotely — Quotation Management

A Go, Gin, GORM, and MySQL quotation manager with a responsive admin workspace, authentication, customer and quotation management, server-calculated totals, and downloadable PDFs.

## Run locally

Requirements: Go 1.25+ and MySQL 8 (or compatible).

1. Create a MySQL database, for example `quotation`.
2. Set database credentials in `.env` (see `.env.example`).
3. Run `go run .` and open `http://localhost:8080/`.
4. On first visit, create the workspace administrator. The initial setup endpoint is disabled after the first account is created.

The app checks the database connection and automatically migrates its schema on startup. Override the listen port with `APP_ADDRESS=:8081`.

To develop with Air live reload, run `air` from the repository root.

## Workspace features

- Dashboard quotation counts (total, pending, approved, rejected) and recent activity.
- Customer create, search, edit, view, pagination, and delete.
- Quotation create, search, status filtering, edit, item management, calculated totals, preview, print, PDF download, and delete.
- Role-based access for administrators, Sales Executives, and Managers. Sales Executives manage customers and create/edit quotations. Managers review pending quotations and approve or reject them. Administrators manage all records and team access.
- Responsive sidebar, navigation, forms, lists, and quotation preview.
- Browser assets are maintained separately in `web/index.html`, `web/styles.css`, and `web/app.js`; role permissions are centralized in `middleware/permissions.go`.

## Roles

| Role | Customers | Quotations | Approvals | Team accounts |
| --- | --- | --- | --- | --- |
| Administrator | Full access | Full access | Full access | Create and remove |
| Sales Executive | Create, view, edit, delete | Create, view, edit | None | None |
| Manager | No access | Review and view | Approve or reject pending quotations | None |

Administrators assign Sales Executive or Manager roles from **Team members**. Existing `user` accounts are treated as Sales Executives for compatibility.

## Authentication

The first user created through the browser setup flow is the administrator. Users sign in with email and password. Passwords are bcrypt-hashed; random bearer tokens are stored in the database as SHA-256 hashes and expire after 24 hours. Logout revokes the active token. Team-member management is administrator-only, and the final administrator cannot be removed.

## API

JSON endpoints are prefixed with `/api/v1`. Except for setup status, first-time setup, and login, endpoints require `Authorization: Bearer <token>`. Responses use a `data` object; paginated lists also include `meta`. Errors use `{"error":"..."}`.

| Method | Path | Purpose |
| --- | --- | --- |
| `GET` | `/auth/setup` | Check whether first-time setup is required |
| `POST` | `/auth/setup` | Create the first administrator |
| `POST` | `/auth/login` | Sign in |
| `GET` | `/auth/me` | Get the current account |
| `POST` | `/auth/logout` | Revoke the current session |
| `GET` | `/dashboard` | Dashboard statistics and recent quotations |
| `GET`, `POST` | `/customers` | Search/list and create customers |
| `GET`, `PUT`, `DELETE` | `/customers/:id` | View, edit, and delete a customer |
| `GET`, `POST` | `/quotations` | Search/filter/list and create quotations |
| `GET`, `PUT`, `DELETE` | `/quotations/:id` | View, edit, and delete a quotation |
| `PATCH` | `/quotations/:id/status` | Update status |
| `GET` | `/quotations/:id/pdf` | Download quotation PDF |
| `GET`, `POST` | `/users` | List and create accounts (admin only) |
| `DELETE` | `/users/:id` | Remove an account (admin only) |

Quotation creation/update JSON example:

```json
{
  "customer_id": 1,
  "tax_rate": 8.25,
  "notes": "Payment due within 30 days.",
  "valid_until": "2026-12-31T00:00:00Z",
  "items": [
    { "description": "Consulting", "quantity": 3, "unit_price": 125.00 },
    { "description": "Setup", "quantity": 1, "unit_price": 80.00 }
  ]
}
```

New quotations start in `draft`. Supported statuses are `draft`, `sent`, `accepted`, `rejected`, and `expired`. Monetary values are rounded to two decimal places. List endpoints support `page`, `page_size` (maximum 100), and `search`; quotations also support a `status` filter.
