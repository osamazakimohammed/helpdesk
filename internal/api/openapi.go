package api

import (
	_ "embed"
	"net/http"
)

const OpenAPIJSON = `{
  "openapi": "3.1.0",
  "info": {
    "title": "Helpdesk Support Platform API",
    "version": "1.0.0",
    "description": "Comprehensive REST API for Enterprise Open-Source Support Ticketing Platform."
  },
  "servers": [
    {
      "url": "/api/v1",
      "description": "Primary API Gateway"
    }
  ],
  "paths": {
    "/auth/login": {
      "post": {
        "summary": "Agent / User Login",
        "tags": ["Auth"],
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": {
                "type": "object",
                "required": ["email", "password"],
                "properties": {
                  "email": { "type": "string", "format": "email" },
                  "password": { "type": "string" }
                }
              }
            }
          }
        },
        "responses": {
          "200": { "description": "Successful login with JWT token and user profile" },
          "401": { "description": "Invalid credentials" }
        }
      }
    },
    "/app/tickets": {
      "get": {
        "summary": "List Tickets with Keyset Pagination (Exactly 1 Query)",
        "tags": ["Agent Workspace"],
        "parameters": [
          { "name": "limit", "in": "query", "schema": { "type": "integer", "default": 25 } },
          { "name": "after_updated_at", "in": "query", "schema": { "type": "string", "format": "date-time" } },
          { "name": "after_id", "in": "query", "schema": { "type": "string", "format": "uuid" } },
          { "name": "status_category", "in": "query", "schema": { "type": "string", "enum": ["open", "pending", "paused", "resolved", "closed"] } },
          { "name": "assigned_agent_id", "in": "query", "schema": { "type": "string", "format": "uuid" } },
          { "name": "assigned_team_id", "in": "query", "schema": { "type": "string", "format": "uuid" } },
          { "name": "search", "in": "query", "schema": { "type": "string" } }
        ],
        "responses": {
          "200": { "description": "Array of tickets with aggregated tags and associations" }
        }
      },
      "post": {
        "summary": "Create Ticket (Agent Workspace)",
        "tags": ["Agent Workspace"],
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": {
                "type": "object",
                "required": ["subject", "description", "contact_email"],
                "properties": {
                  "subject": { "type": "string" },
                  "description": { "type": "string" },
                  "contact_email": { "type": "string", "format": "email" },
                  "contact_name": { "type": "string" },
                  "priority_id": { "type": "string", "format": "uuid" },
                  "type_id": { "type": "string", "format": "uuid" },
                  "assigned_team_id": { "type": "string", "format": "uuid" },
                  "assigned_agent_id": { "type": "string", "format": "uuid" },
                  "tags": { "type": "array", "items": { "type": "string" } }
                }
              }
            }
          }
        },
        "responses": {
          "201": { "description": "Ticket successfully created" }
        }
      }
    },
    "/app/tickets/{id}": {
      "get": {
        "summary": "Get Ticket Detail (Query 1 of 2)",
        "tags": ["Agent Workspace"],
        "parameters": [
          { "name": "id", "in": "path", "required": true, "schema": { "type": "string", "format": "uuid" } }
        ],
        "responses": {
          "200": { "description": "Full Ticket metadata, contact, org, SLA policy, watchers, and links" }
        }
      },
      "patch": {
        "summary": "Update Ticket Fields (Status, Assignee, Priority, SLA)",
        "tags": ["Agent Workspace"],
        "parameters": [
          { "name": "id", "in": "path", "required": true, "schema": { "type": "string", "format": "uuid" } }
        ],
        "responses": {
          "200": { "description": "Updated Ticket" }
        }
      }
    },
    "/app/tickets/{id}/events": {
      "get": {
        "summary": "List Ticket Timeline Events (Query 2 of 2 - Keyset Paginated)",
        "tags": ["Agent Workspace"],
        "parameters": [
          { "name": "id", "in": "path", "required": true, "schema": { "type": "string", "format": "uuid" } },
          { "name": "limit", "in": "query", "schema": { "type": "integer", "default": 50 } },
          { "name": "visibility", "in": "query", "schema": { "type": "string", "enum": ["public", "internal"] } }
        ],
        "responses": {
          "200": { "description": "List of events with embedded attachments and mentions" }
        }
      },
      "post": {
        "summary": "Add Reply or Internal Note",
        "tags": ["Agent Workspace"],
        "parameters": [
          { "name": "id", "in": "path", "required": true, "schema": { "type": "string", "format": "uuid" } }
        ],
        "responses": {
          "201": { "description": "Event created and queued to outbox" }
        }
      }
    },
    "/portal/tickets": {
      "get": {
        "summary": "Customer Portal Ticket List",
        "tags": ["Customer Portal"],
        "responses": { "200": { "description": "Customer's tickets" } }
      }
    },
    "/kb/articles/{slug}": {
      "get": {
        "summary": "Get Knowledge Base Article by Slug",
        "tags": ["Knowledge Base"],
        "responses": { "200": { "description": "Published article with HTML and metadata" } }
      }
    },
    "/submit/ticket": {
      "post": {
        "summary": "Submit Anonymous Support Ticket",
        "tags": ["Anonymous Intake"],
        "responses": { "201": { "description": "Ticket created and reference token returned" } }
      }
    }
  }
}`

func HandleOpenAPISpec(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(OpenAPIJSON))
}

func HandleSwaggerUI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	html := `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <title>Helpdesk API Documentation</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css" />
</head>
<body style="margin:0;padding:0;background:#0f172a;">
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js" crossorigin></script>
  <script>
    window.onload = () => {
      window.ui = SwaggerUIBundle({
        url: '/api/v1/openapi.json',
        dom_id: '#swagger-ui',
        deepLinking: true,
        presets: [SwaggerUIBundle.presets.apis],
      });
    };
  </script>
</body>
</html>`
	_, _ = w.Write([]byte(html))
}
