import { describe, expect, it } from "vitest";
import { validateDescription, validateName } from "./validation";

describe("validateName", () => {
  it.each(["payments-api", "abc", "a1-b2", "123"])("accepts %s", (name) => {
    expect(validateName(name)).toBeUndefined();
  });

  it.each(["ab", "a".repeat(101), "Payments", "-payments", "payments-", "pay_ments", "pay ments"])(
    "rejects %s",
    (name) => {
      expect(validateName(name)).toBeDefined();
    },
  );
});

describe("validateDescription", () => {
  it("limits the length", () => {
    expect(validateDescription("x".repeat(350))).toBeUndefined();
    expect(validateDescription("x".repeat(351))).toBeDefined();
  });
});
