# Quotation Management API

A beginner-friendly REST API built with Go, Gin, GORM, and MySQL. It manages customers and quotations, calculates line totals, subtotal, tax, and grand total on the server, updates quotation statuses, and exports quotations as PDFs.

## Requirements

- Go 1.25 or newer
- MySQL 8 or compatible

## Run locally

1. Create a MySQL database, for example `quotation`.
2. Copy `.env.example` to `.env` and set your database credentials.
3. Start the API:

   ```powershell
   go mod tidy
   go run .
   ```

The application checks the database connection, runs GORM `AutoMigrate`, and listens on `http://localhost:8080` by default. Configuration is read from `.env` or the process environment.

Open `http://localhost:8080/` for the built-in browser interface. The JSON API is available at `http://localhost:8080/api/v1`.

## API

All endpoints are prefixed with `/api/v1`. JSON endpoints return a `data` object (or list) and list endpoints include `meta` pagination information. Errors return `{"error":"..."}`.

| Method | Path | Purpose |
| --- | --- | --- |
| `POST` | `/customers` | Create a customer |
| `GET` | `/customers?page=1&page_size=20` | List customers |
| `GET` | `/customers/:id` | Get a customer |
| `PUT` | `/customers/:id` | Replace customer fields |
| `DELETE` | `/customers/:id` | Delete a customer |
| `POST` | `/quotations` | Create a quotation with items |
| `GET` | `/quotations?page=1&page_size=20` | List quotations |
| `GET` | `/quotations/:id` | Get a quotation |
| `PUT` | `/quotations/:id` | Replace quotation details and items |
| `PATCH` | `/quotations/:id/status` | Change quotation status |
| `DELETE` | `/quotations/:id` | Delete a quotation |
| `GET` | `/quotations/:id/pdf` | Download a quotation PDF |

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

The API generates the quotation number and starts new quotations in `draft` status. Supported statuses are `draft`, `sent`, `accepted`, `rejected`, and `expired`. Monetary values are rounded to two decimal places. Pagination defaults to 20 records and is capped at 100.
