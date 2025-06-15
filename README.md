# **PayFlow - Automated Payroll Platform**

[![Go Report Card](https://goreportcard.com/badge/github.com/your-username/payflow)](https://goreportcard.com/report/github.com/your-username/payflow)
[![CI](https://github.com/your-username/payflow/actions/workflows/ci.yml/badge.svg)](https://github.com/your-username/payflow/actions/workflows/ci.yml)

PayFlow is a robust, production-grade backend service designed to simplify and automate financial operations for Small to Medium-sized Enterprises (SMEs). The core of the platform is a powerful payroll automation engine that handles everything from salary calculation to bulk disbursement.

This repository contains the backend service, built with Go, following Clean Architecture principles for maximum maintainability, scalability, and testability.

## **Table of Contents**

1.  [Features & Implementation Status](#features--implementation-status)
2.  [Architecture](#architecture)
3.  [Tech Stack](#tech-stack)
4.  [Getting Started](#getting-started)
    *   [Prerequisites](#prerequisites)
    *   [Local Development Setup](#local-development-setup)
5.  [API Endpoints](#api-endpoints)
6.  [Running Migrations](#running-migrations)
7.  [Future Work](#future-work)

---

## **Features & Implementation Status**

This section tracks the progress of the MVP features.

| Feature Area                  | Status                                            | Description                                                                                             |
| ----------------------------- | ------------------------------------------------- | ------------------------------------------------------------------------------------------------------- |
| **Foundational Setup**        | ✅ **Complete**                                   | Project structure, configuration management (`viper`), structured logging (`zerolog`), and Docker setup.  |
| **User & Business Auth**      | ✅ **Complete**                                   | Secure user registration for a new business, JWT-based login, and role-based access control middleware. |
| **Employee Management**       | ✅ **Complete**                                   | Full CRUD (Create, Read, Update, Deactivate) operations for employee records.                           |
| **Cadre (Salary) Management** | 🚧 **In Progress**                                | CRUD operations for managing salary structures (cadres), including earnings and deduction rules.        |
| **Deduction Rule Management** | ⏳ **Pending**                                    | Admin-only endpoints to configure company-wide deduction rules (e.g., taxes, pensions).                 |
| **Payroll Calculation Engine**| ✅ **Complete**                                   | Core service logic can accurately calculate gross pay, deductions, and net pay for all employees.       |
| **Payroll Workflow**          | ✅ **Complete**                                   | Full workflow: Create Run -> Submit for Approval -> Approve/Reject Run.                                 |
| **Payment Disbursement**      | 🚧 **In Progress (KoraPay)**                      | Integration with KoraPay for bulk payment disbursement. Interface is defined, implementation ongoing. |
| **Background Jobs & Scheduler**| ✅ **Complete**                                   | A robust scheduler (`gocron`) handles the execution of approved payrolls on their scheduled date.     |
| **Email Notifications**       | ✅ **Complete (MailHog)**                         | System can send notifications for key events (e.g., payroll rejection). Uses MailHog for local dev.   |

---

## **Architecture**

This project strictly follows **Clean Architecture** principles to ensure a separation of concerns.

-   `internal/domain`: Contains the core business entities and rules. It has no external dependencies.
-   `internal/service`: Orchestrates the business logic and use cases (e.g., `PayrollService`). Depends only on `domain` and repository interfaces.
-   `internal/repository`: Defines interfaces for data persistence (`UserRepository`) and contains concrete implementations (e.g., `postgres`).
-   `internal/platform`: Contains concrete implementations for external services like payment gateways (`korapay`), schedulers (`scheduler`), and notification providers (`sendgrid`).
-   `internal/api`: The delivery layer. It handles HTTP requests, validation, and responses. It depends on the `service` layer to perform actions.

This structure ensures that the core business logic is independent of the database, web framework, and any third-party services, making the system highly adaptable and testable.

---

## **Tech Stack**

-   **Language**: Go (v1.22+)
-   **API Framework**: `chi`
-   **Database**: PostgreSQL
-   **ORM**: `gorm`
-   **Migrations**: `golang-migrate/migrate`
-   **Configuration**: `viper`
-   **Logging**: `zerolog`
-   **Background Jobs**: `gocron`
-   **Local Dev Environment**: Docker & Docker Compose

---

## **Getting Started**

### **Prerequisites**

-   [Go](https://go.dev/doc/install) (v1.22 or later)
-   [Docker](https://www.docker.com/get-started/) and [Docker Compose](https://docs.docker.com/compose/install/)
-   [golang-migrate/migrate](https://github.com/golang-migrate/migrate/tree/master/cmd/migrate) CLI tool.
-   `make` (optional, but recommended for using the Makefile shortcuts)

### **Local Development Setup**

1.  **Clone the repository:**
    ```bash
    git clone https://github.com/your-username/payflow.git
    cd payflow
    ```

2.  **Create your environment file:**
    Copy the example file and fill in your details. For local development, the defaults are usually sufficient.
    ```bash
    cp .env.example .env
    ```

3.  **Start the infrastructure:**
    This command will start the PostgreSQL database and a MailHog instance for testing email notifications.
    ```bash
    make up
    ```
    Alternatively, if not using `make`:
    ```bash
    docker-compose up -d
    ```

4.  **Run database migrations:**
    This will create all the necessary tables in the `payflow_db` database.
    ```bash
    make migrate-up
    ```

5.  **Install dependencies:**
    ```bash
    go mod tidy
    ```

6.  **Run the application:**
    The server will start on `http://localhost:8080`.
    ```bash
    make run
    ```
    Alternatively, if not using `make`:
    ```bash
    go run ./cmd/server/main.go
    ```
    You can now access the API endpoints using a tool like Postman or `curl`.

---

## **API Endpoints**

The API is versioned under `/v1`. For detailed information on request/response formats for each endpoint, please refer to our Postman collection or OpenAPI/Swagger documentation (pending).

**Key Endpoints Implemented:**

-   `POST /v1/auth/register`: Create a new business and admin user.
-   `POST /v1/auth/login`: Log in to receive a JWT.
-   `POST /v1/employees`: Create a new employee (Admin/Operator role required).
-   `GET /v1/employees`: List all employees for the business (Admin/Operator role required).
-   `GET /v1/employees/{id}`: Get a single employee's details.
-   `POST /v1/payroll-runs`: Create a new draft payroll run.
-   `POST /v1/payroll-runs/{id}/submit`: Submit a draft for approval.
-   `POST /v1/payroll-runs/{id}/approve`: Approve a run for payment (Approver/Admin role required).
-   `POST /v1/payroll-runs/{id}/reject`: Reject a run and send it back to draft (Approver/Admin role required).

---

## **Running Migrations**

Database schema changes are managed via raw SQL migration files in the `/migrations` directory. Use the `migrate` CLI tool via the Makefile for safety.

-   **Create a new migration:**
    ```bash
    make migrate-create name=add_new_feature_table
    ```

-   **Apply all pending migrations:**
    ```bash
    make migrate-up
    ```

-   **Revert the last applied migration:**
    ```bash
    make migrate-down
    ```

---

## **Future Work**

-   **Complete Deduction Rule Management**: Implement the service and repository for full CRUD on tax/pension rules.
-   **Finalize KoraPay Integration**: Complete the `payout.go` client to handle real-world disbursement calls and error handling.
-   **Implement Unit & Integration Tests**: Add comprehensive test coverage for services and handlers.
-   **Generate API Documentation**: Create an OpenAPI (Swagger) specification for the API.
-   **Enhance Security**: Implement more robust security measures like refresh tokens, detailed audit logging, and rate limiting.