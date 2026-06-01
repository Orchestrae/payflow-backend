// tests/load/payroll_load.js
// k6 load test for PayFlow API
// Run: k6 run tests/load/payroll_load.js
// Override base URL: K6_BASE_URL=http://your-host:8080 k6 run tests/load/payroll_load.js

import http from "k6/http";
import { check, sleep, group } from "k6";
import { Trend, Counter, Rate } from "k6/metrics";

// Custom metrics
const loginDuration = new Trend("login_duration", true);
const employeeCreateDuration = new Trend("employee_create_duration", true);
const payrollCreateDuration = new Trend("payroll_create_duration", true);
const depositDuration = new Trend("deposit_duration", true);
const apiErrors = new Counter("api_errors");
const successRate = new Rate("success_rate");

// Configuration
const BASE_URL = __ENV.K6_BASE_URL || "http://localhost:8080";

export const options = {
  stages: [
    { duration: "5s", target: 20 },
    { duration: "20s", target: 100 },
    { duration: "5s", target: 0 },
  ],
  thresholds: {
    http_req_duration: ["p(95)<2000"],
    success_rate: ["rate>0.9"],
    api_errors: ["count<50"],
  },
};

const headers = {
  "Content-Type": "application/json",
};

function registerAndLogin(vuId) {
  const email = "loadtest_vu" + vuId + "_" + Date.now() + "@example.com";
  const password = "LoadTest@123";

  const registerPayload = JSON.stringify({
    business_name: "LoadTest Corp " + vuId,
    email: email,
    password: password,
    rc_number: "RC" + String(vuId).padStart(6, "0"),
    incorporation_date: "2020-01-01T00:00:00Z",
    director_bvn: "12345678901",
  });

  http.post(BASE_URL + "/v1/auth/register", registerPayload, {
    headers: headers,
    tags: { name: "Register" },
  });

  const loginPayload = JSON.stringify({
    email: email,
    password: password,
  });

  const start = Date.now();
  const loginRes = http.post(BASE_URL + "/v1/auth/login", loginPayload, {
    headers: headers,
    tags: { name: "Login" },
  });
  loginDuration.add(Date.now() - start);

  const loginOk = check(loginRes, {
    "login status is 200": function (r) { return r.status === 200; },
    "login returns token": function (r) {
      try {
        var body = JSON.parse(r.body);
        return body.token !== undefined && body.token !== "";
      } catch (e) {
        return false;
      }
    },
  });

  if (!loginOk) {
    apiErrors.add(1);
    successRate.add(false);
    return null;
  }

  successRate.add(true);

  try {
    var body = JSON.parse(loginRes.body);
    return body.token;
  } catch (e) {
    return null;
  }
}

function employeeCRUD(token) {
  if (!token) return;

  var authHeaders = {
    "Content-Type": "application/json",
    Authorization: "Bearer " + token,
  };

  group("Employee CRUD", function () {
    var cadrePayload = JSON.stringify({
      name: "Cadre " + Date.now(),
      earning_components: [
        { name: "Basic Salary", amount: 30000000, component_type: "basic" },
        { name: "Housing", amount: 12000000, component_type: "housing" },
        { name: "Transport", amount: 8000000, component_type: "transport" },
      ],
    });

    var cadreRes = http.post(BASE_URL + "/v1/cadres", cadrePayload, {
      headers: authHeaders,
      tags: { name: "CreateCadre" },
    });

    var cadreId;
    try {
      var cadreBody = JSON.parse(cadreRes.body);
      cadreId = cadreBody.data ? cadreBody.data.id : cadreBody.id;
    } catch (e) {
      apiErrors.add(1);
      return;
    }

    if (!cadreId) {
      apiErrors.add(1);
      return;
    }

    var empPayload = JSON.stringify({
      full_name: "Load Test Employee " + Date.now(),
      email: "emp_" + Date.now() + "_" + Math.random().toString(36).slice(2) + "@test.com",
      cadre_id: cadreId,
      bank_name: "Test Bank",
      bank_code: "058",
      bank_account_number: "" + Math.floor(1000000000 + Math.random() * 9000000000),
    });

    var start = Date.now();
    var empRes = http.post(BASE_URL + "/v1/employees", empPayload, {
      headers: authHeaders,
      tags: { name: "CreateEmployee" },
    });
    employeeCreateDuration.add(Date.now() - start);

    var empOk = check(empRes, {
      "create employee status is 201 or 200": function (r) {
        return r.status === 201 || r.status === 200;
      },
    });

    if (empOk) {
      successRate.add(true);
    } else {
      apiErrors.add(1);
      successRate.add(false);
    }

    var listRes = http.get(BASE_URL + "/v1/employees", {
      headers: authHeaders,
      tags: { name: "ListEmployees" },
    });

    check(listRes, {
      "list employees status is 200": function (r) { return r.status === 200; },
    });
  });
}

function payrollCreation(token) {
  if (!token) return;

  var authHeaders = {
    "Content-Type": "application/json",
    Authorization: "Bearer " + token,
  };

  group("Payroll Creation", function () {
    var payrollPayload = JSON.stringify({
      period: "2026-06-01T00:00:00Z",
      adjustments: {},
    });

    var start = Date.now();
    var payrollRes = http.post(BASE_URL + "/v1/payroll-runs", payrollPayload, {
      headers: authHeaders,
      tags: { name: "CreatePayroll" },
    });
    payrollCreateDuration.add(Date.now() - start);

    var ok = check(payrollRes, {
      "payroll creation returns 200 or 201": function (r) {
        return r.status === 200 || r.status === 201;
      },
    });

    if (ok) {
      successRate.add(true);
    } else {
      successRate.add(false);
    }

    var listRes = http.get(BASE_URL + "/v1/payroll-runs", {
      headers: authHeaders,
      tags: { name: "ListPayrollRuns" },
    });

    check(listRes, {
      "list payroll runs status is 200": function (r) { return r.status === 200; },
    });
  });
}

function depositInitiation(token) {
  if (!token) return;

  var authHeaders = {
    "Content-Type": "application/json",
    Authorization: "Bearer " + token,
  };

  group("Deposit Initiation", function () {
    var depositPayload = JSON.stringify({
      amount: 100000,
      currency: "NGN",
    });

    var start = Date.now();
    var depositRes = http.post(BASE_URL + "/v1/wallets/deposit", depositPayload, {
      headers: authHeaders,
      tags: { name: "InitiateDeposit" },
    });
    depositDuration.add(Date.now() - start);

    var ok = check(depositRes, {
      "deposit returns 200 or 201 or 400": function (r) {
        return r.status === 200 || r.status === 201 || r.status === 400;
      },
    });

    if (ok) {
      successRate.add(true);
    } else {
      apiErrors.add(1);
      successRate.add(false);
    }

    var balanceRes = http.get(BASE_URL + "/v1/wallets/balance", {
      headers: authHeaders,
      tags: { name: "GetBalance" },
    });

    check(balanceRes, {
      "balance check returns 200 or 404": function (r) {
        return r.status === 200 || r.status === 404;
      },
    });
  });
}

export default function () {
  var vuId = __VU;

  var token = registerAndLogin(vuId);

  if (token) {
    employeeCRUD(token);
    sleep(0.5);

    payrollCreation(token);
    sleep(0.5);

    depositInitiation(token);
    sleep(0.5);
  } else {
    sleep(1);
  }
}
