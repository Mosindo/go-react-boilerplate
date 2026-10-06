import assert from "node:assert/strict";
import { test } from "node:test";
import { describeError } from "./errors.ts";

test("describeError maps known API messages to French", () => {
  assert.equal(describeError({ status: 401, message: "invalid credentials" }), "Votre session a expiré. Reconnectez-vous.");
  assert.equal(describeError({ status: 409, message: "email already exists" }), "Un compte existe déjà avec cet e-mail.");
  assert.equal(describeError({ status: 422, message: "you must be at least 18 years old" }), "Vous devez avoir 18 ans ou plus.");
});

test("describeError handles network, rate limit and server errors", () => {
  assert.match(describeError({ name: "NetworkError", message: "x" }), /Connexion impossible/);
  assert.match(describeError({ status: 429, message: "too many requests" }), /Trop de tentatives/);
  assert.match(describeError({ status: 503, message: "boom" }), /problème/);
  assert.equal(describeError({ status: 403, code: "profile_incomplete", message: "x" }), "Terminez d'abord votre profil.");
});

test("describeError never leaks raw technical messages", () => {
  assert.equal(
    describeError({ status: 400, message: "Key: 'RegisterRequest.Email' Error:Field validation" }),
    "Une erreur est survenue. Réessayez."
  );
  assert.equal(describeError(undefined, "fallback"), "fallback");
});

test("describeError explains suspended accounts", () => {
  assert.match(describeError({ status: 403, code: "account_suspended", message: "account suspended" }), /suspendu/);
});
