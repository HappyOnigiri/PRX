import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { PullRequestFlags } from "../src/views/PullRequestFlags";

describe("PullRequestFlags", () => {
  it("gives the reason on hover and on focus, then takes it away", () => {
    render(<PullRequestFlags stale syncError="offline" />);
    const stale = screen.getByRole("button", {
      name: "Stale: this pull request may no longer match its state on GitHub.",
    });
    const syncError = screen.getByRole("button", {
      name: "GitHub sync error: offline",
    });
    expect(document.querySelector(".pr-flag-tip")).toBeNull();

    fireEvent.mouseEnter(stale);
    expect(document.querySelector(".pr-flag-tip")).toHaveTextContent("Stale:");
    fireEvent.mouseLeave(stale);
    expect(document.querySelector(".pr-flag-tip")).toBeNull();

    // 理由はポインタだけのものにしない。keyboard で辿り着いても同じものが出る。
    fireEvent.focus(syncError);
    expect(document.querySelector(".pr-flag-tip")).toHaveTextContent("offline");
    fireEvent.blur(syncError);
    expect(document.querySelector(".pr-flag-tip")).toBeNull();
  });

  it("stays silent while the pull request has nothing wrong with it", () => {
    const { container } = render(
      <PullRequestFlags stale={false} syncError="" />,
    );
    expect(container.querySelector(".pr-flag")).toBeNull();
  });
});
