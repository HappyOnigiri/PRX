import { describe, expect, it } from "vitest";
import { formValue } from "../src/form";

describe("formValue", () => {
  it("returns text values and rejects missing or file values", () => {
    const data = new FormData();
    data.set("title", "Release PRX");
    data.set("attachment", new File(["content"], "notes.txt"));

    expect(formValue(data, "title")).toBe("Release PRX");
    expect(formValue(data, "attachment")).toBe("");
    expect(formValue(data, "missing")).toBe("");
  });
});
