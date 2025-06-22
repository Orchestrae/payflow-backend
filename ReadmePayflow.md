# PayFlow API Documentation

This document provides a detailed overview of the PayFlow API endpoints, their functionality, expected payloads, and authorization requirements. The base URL for all endpoints is `/v1`.

## **Table of Contents**

1.  [Authentication](#authentication)
2.  [Employee Management](#employee-management)
3.  [Cadre (Salary Structure) Management](#cadre-salary-structure-management)
4.  [Payroll Workflow](#payroll-workflow)
5.  [Background Jobs & Scheduler](#background-jobs--scheduler)

---

## **1. Authentication**

These endpoints manage user and business onboarding and access to the platform.

### `POST /auth/register`

-   **Description:** Creates a new Business and its primary Admin user in a single, atomic transaction. This is the first step for any new company joining PayFlow.
-   **How it Works:**
    1.  Accepts the desired `business_name`, `email`, and `password` for the admin.
    2.  The `AuthService` initiates a database transaction.
    3.  A new `Business` record is created.
    4.  A new `User` record is created with the 'Admin' role and linked to the new business.
    5.  The transaction is committed. If any step fails, the entire operation is rolled back to prevent orphaned data.
    6.  A success response is returned. An email verification flow would typically be triggered here.
-   **Authorization:** Public (No token required).
-   **Example Payload:**
    ```json
    {
      "business_name": "Innovate Inc.",
      "email": "admin@innovate.com",
      "password": "a-very-strong-password"
    }
    ```

### `POST /auth/login`

-   **Description:** Authenticates a user and returns a JSON Web Token (JWT) for accessing protected endpoints.
-   **How it Works:**
    1.  The user provides their `email` and `password`.
    2.  The `AuthService` finds the user by email and securely compares the provided password with the stored hash.
    3.  If credentials are valid, a new JWT is generated containing the `user_id`, `business_id`, and `role`.
    4.  The token is returned to the client, who must include it in the `Authorization: Bearer <token>` header for all subsequent protected requests.
-   **Authorization:** Public.
-   **Example Payload:**
    ```json
    {
      "email": "admin@innovate.com",
      "password": "a-very-strong-password"
    }
    ```

---

## **2. Employee Management**

Endpoints for managing the employee records within a business.

### `POST /employees`

-   **Description:** Adds a new employee to the business.
-   **How it Works:** An Admin or Operator provides the employee's details. The service validates that the specified `cadre_id` exists and belongs to the same business before creating the employee record.
-   **Authorization:** `Admin` or `Operator` role required.
-   **Example Payload:**
    ```json
    {
      "full_name": "Jane Doe",
      "email": "jane.doe@innovate.com",
      "cadre_id": 1,
      "bank_name": "Metropolis Bank",
      "bank_account_number": "1234567890"
    }
    ```

### `GET /employees`

-   **Description:** Retrieves a list of all employees associated with the authenticated user's business.
-   **How it Works:** The service queries the database for all employees matching the `business_id` from the user's JWT claims.
-   **Authorization:** `Admin` or `Operator` role required.

### `GET /employees/{id}`

-   **Description:** Fetches the details of a single employee.
-   **How it Works:** The service retrieves the employee by their ID and performs a crucial security check to ensure the employee's `business_id` matches the one in the user's JWT. This prevents users from one company from accessing data from another.
-   **Authorization:** `Admin` or `Operator` role required.

---

## **3. Cadre (Salary Structure) Management**

Endpoints for defining the standardized salary structures of the business.

### `POST /cadres`

-   **Description:** Creates a new salary structure (e.g., "Senior Engineer," "Marketing Manager").
-   **How it Works:** An Admin or Operator defines a cadre with a name, a list of fixed earning components (like Basic Salary, Housing), and links to pre-defined deduction rules. This allows for rapid and consistent employee setup.
-   **Authorization:** `Admin` or `Operator` role required.
-   **Example Payload:**
    ```json
    {
      "name": "Senior Software Engineer",
      "earning_components": [
        { "name": "Basic Salary", "amount": 500000 },
        { "name": "Housing Allowance", "amount": 150000 }
      ],
      "deduction_rule_ids": [1, 2]
    }
    ```

### `GET /cadres`

-   **Description:** Lists all salary cadres configured for the business.
-   **How it Works:** Retrieves all cadre records matching the `business_id` from the user's JWT.
-   **Authorization:** `Admin` or `Operator` role required.

---

## **4. Payroll Workflow**

These endpoints orchestrate the end-to-end process of running payroll.

### `POST /payroll-runs`

-   **Description:** Initiates a new payroll run for the current period.
-   **How it Works:**
    1.  An Operator or Admin triggers this endpoint, optionally providing one-time adjustments (bonuses or deductions) for specific employees.
    2.  The `PayrollService` fetches all active employees and their cadre information.
    3.  The core calculation engine computes the gross pay, deductions, and net pay for every employee.
    4.  A new `PayrollRun` record is created and saved to the database with the status `draft`.
-   **Authorization:** `Admin` or `Operator` role required.
  -   **Example Payload:**
      ```json
      {
        "adjustments": {
          "15": 5000,   // Bonus of 5000 for employee with ID 15
          "22": -2000   // Deduction of 2000 for employee with ID 22
        }
      }
      ```
      {
      "payroll_date": "2025-06-25",
      "adjustments": {
      "101": 50000,   // A positive value for Alice's bonus ($500.00)
      "102": -7500    // A negative value for Bob's deduction ($75.00)
        }
     }
    ```

### `POST /payroll-runs/{id}/submit`

-   **Description:** Submits a `draft` payroll run for review and approval.
-   **How it Works:** An Operator, after verifying the draft payroll, calls this endpoint. The service changes the run's status from `draft` to `pending_approval` and triggers an email notification to all users with the 'Approver' role in the business.
-   **Authorization:** `Admin` or `Operator` role required.

### `POST /payroll-runs/{id}/approve`

-   **Description:** Provides the final sign-off for a payroll run, scheduling it for payment.
-   **How it Works:**
    1.  An Approver or Admin, after reviewing the itemized breakdown, calls this endpoint.
    2.  The service validates the run is in the `pending_approval` state.
    3.  The run's status is changed to `approved`.
    4.  The `PayrollService` then calls the `PayoutScheduler`, telling it to create a background job to execute the disbursement on the `scheduled_for` date.
-   **Authorization:** `Admin` or `Approver` role required.

### `POST /payroll-runs/{id}/reject`

-   **Description:** Rejects a payroll run, sending it back to the Operator for corrections.
-   **How it an Works:** An Approver or Admin provides a `reason` for the rejection. The service changes the run's status to `rejected`, saves the reason, and sends a notification email back to the Operator detailing why the run was rejected, allowing them to fix and resubmit it.
-   **Authorization:** `Admin` or `Approver` role required.
-   **Example Payload:**
    ```json
    {
      "reason": "Bonus calculation for the sales team is incorrect. Please revise."
    }
    ```
---

## **5. Background Jobs & Scheduler**

This is not a direct API endpoint but a crucial backend process that underpins the automation.

-   **Description:** The scheduler is a system that manages and executes time-based tasks without direct user interaction.
-   **How it Works in PayFlow:**
    1.  When a payroll run is approved via `POST /payroll-runs/{id}/approve`, the `PayrollService` schedules a one-time job with our `gocron` scheduler. The job is set to trigger on the run's `scheduled_for` date.
    2.  On the scheduled date, the scheduler wakes up and executes the job.
    3.  The job calls the `PayrollService.ProcessApprovedPayroll` method.
    4.  This method updates the run's status to `processing`, calls the KoraPay integration to perform the bulk payment, and upon success, updates the status to `completed`. If the payment fails, it updates the status to `failed` and notifies the admin.

This decoupling ensures that the API response for approving a payroll is instant, while the actual, potentially long-running, payment process happens reliably in the background.