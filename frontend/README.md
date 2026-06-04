# Gamero Frontend

Gamero is a game creator collaboration platform built with React, TypeScript, Vite, and TanStack Query.

## Core Modules

The frontend exposes six core functional modules with end-to-end pages and API integration:

| Module | Create | Read | Update | Delete / Close |
| --- | --- | --- | --- | --- |
| Users and profiles | Email registration, portfolio items | Profile pages, followers, talents | Profile, avatar, skills, availability, portfolio | Account deletion, portfolio deletion |
| Projects | Create project | Project list/detail, releases, members | Edit project, status, media, members | Delete project, remove members |
| Dev logs | Create draft/release log | Log list/detail, comments | Edit log, publish, media | Delete log, delete comments/media |
| Community posts | Create post modal | Topic list, post list/detail, comments | Edit post, update comments | Delete post, delete comments |
| Recruitment and collaboration | Create recruitment, apply, invite talent | Recruitment list/detail, applications, invitations | Edit recruitment, close/reopen, approve/reject applications | Delete recruitment, withdraw applications |
| Admin console | Create topics, sensitive words | Stats, users, reports, content, audit logs | User roles, report status, topics, bans | Delete topics, users, posts, comments, sensitive words |

Supporting capabilities include notifications, WebSocket updates, search, reporting, file uploads, JWT refresh, and RBAC-protected admin APIs.

## Development

The dev server is pinned to port `5714`.

```bash
npm run dev
```

Open:

```text
http://localhost:5714
```

## Admin Login

Use the seeded/admin database account:

```text
Email: admin@gamero.local
Password: Gamero123
```

Admin and superadmin users are redirected to `/admin` after login.

## Build

```bash
npm run build
```
