# School Management Platform (Go + Templ + HTMX + BoltDB)

A lightweight school management platform built in pure Go, designed for managing users, classes, assignments, and submissions. The project focuses on simplicity, performance, and clean architecture without external frameworks.

## Overview

This application provides:

- User authentication with password hashing and encrypted password storage.

- Role-based access (student and professor).

- Class management with multiple users per class.

- Assignment management per class.

- Submission tracking per assignment and student.

- Dynamic frontend rendering using Templ and HTMX.

- Persistent storage using bbolt (BoltDB).

- The system is intentionally built with minimal dependencies and a clear separation between database models, DTOs, and presentation logic.

## Technology Stack

Backend:

- Go (net/http)

- bbolt (embedded key-value database)

- Generic database wrappers

- AES encryption for stored plaintext passwords (administrative use)

- Password hashing for authentication

Frontend:

- Templ (type-safe Go HTML templates)

- HTMX (partial page updates)

- Tailwind CSS

- DaisyUI

## Architecture
### Database Layer

The database uses bbolt with structured buckets:

- Users

- Subjects

-  Classes

- Assignments

- Submissions

Data is stored using composite keys where necessary. For example:

Submissions use a composite key:
```
assignmentId:username
```
This ensures uniqueness per assignment and user while allowing efficient prefix scans.

Generic helpers are implemented for:

- Save

- Get

- Exists

- ListByPrefix

- GetWithPrefix

All database models are located under:
```
/database/models
```
### DTO Layer

DTOs are separated from database models to prevent coupling between storage and presentation.

Located under:
```
/dto
```
Conversion functions are used to map models to DTOs explicitly. This allows future divergence between database structures and UI representations.

### Routing

Routing is handled manually using net/http and a centralized router function. Boolean switch statements are used for flexible route matching.

Examples of supported routes:

- /login

- /

- /{classId}/asignaciones

Authentication is enforced via HTTP cookies. All routes except /login require a valid session cookie.

### Authentication

Authentication flow:

1. User submits login form (HTMX POST).

2. Password is verified using a hashed comparison.

3. On success:

- A session cookie is set.

- HTMX redirects to /.

4. On failure:

- Error message is returned as HTML fragment.

- Displayed dynamically without page reload.

Passwords are:

- Hashed for verification.

- Encrypted (AES) for administrative recovery scenarios.

### Assignment and Submission Logic

Assignments:

- Belong to one class.

- Stored with auto-incremented IDs.

- Retrieved using class-based prefix scanning.

Submissions:

- Unique per assignment and username.

- Stored using composite key: assignmentId:username.

- Support listing by assignment via prefix.

- Support listing by user (currently filtered in-memory).

Designed to scale comfortably for:

- Up to 200 students per assignment.

- Thousands of total submissions.

## Development

To run in development mode:
```
goplate watch
```
This will:

- Rebuild Tailwind CSS

- Regenerate templ components

- Restart the server

Server runs at:
```
http://localhost:3000
```
If styles appear broken after changes, perform a hard refresh due to browser caching.

## Production Considerations

- Enable Secure: true on cookies when using HTTPS.

- Implement CSS cache-busting (hash or version query parameter).

- Consider adding a secondary index bucket if submission volume grows significantly.

- Replace username-based session cookie with a random session token for stronger security.

# Project Goals

This project prioritizes:

- Clean architecture

- Minimal dependencies

- Explicit data transformations

- Predictable routing

- Embedded database simplicity

- Full control over the stack

It is designed as a learning-focused but production-capable foundation for a school management system.

## License

Apache 2.0
