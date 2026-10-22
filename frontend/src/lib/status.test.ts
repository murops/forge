import { describe, expect, it } from "vitest";
import { isFinished } from "./status";

describe("isFinished", () => {
  it("is true only for final states", () => {
    expect(isFinished("pending")).toBe(false);
    expect(isFinished("running")).toBe(false);
    expect(isFinished("succeeded")).toBe(true);
    expect(isFinished("failed")).toBe(true);
  });
});
