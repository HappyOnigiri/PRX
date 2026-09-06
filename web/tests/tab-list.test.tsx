import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { useState } from "react";
import { afterEach, describe, expect, it } from "vitest";
import { TabList, TabPanel } from "../src/views/TabList";

const tabs = [
  { id: "one", label: "One" },
  { id: "two", label: "Two" },
  { id: "three", label: "Three" },
] as const;

type Id = (typeof tabs)[number]["id"];

function Harness({ focusOnMount }: { focusOnMount?: boolean }) {
  const [active, setActive] = useState<Id>("one");
  return (
    <>
      <TabList
        tabs={tabs}
        active={active}
        onSelect={setActive}
        idPrefix="demo"
        className="demo-tabs"
        tabClassName="demo-tab"
        label="Demo tabs"
        {...(focusOnMount === undefined ? {} : { focusOnMount })}
      />
      {tabs.map((tab) => (
        <TabPanel
          key={tab.id}
          active={tab.id === active}
          className="demo-panel"
          idPrefix="demo"
          tab={tab.id}
        >
          {tab.label} panel
        </TabPanel>
      ))}
    </>
  );
}

describe("TabList", () => {
  afterEach(cleanup);

  // One tab stop enters the strip and the arrow keys move within it, which is
  // the pattern the tab role obliges the strip to implement.
  it("keeps a single tab stop and moves the selection with the keyboard", () => {
    render(<Harness />);
    const one = screen.getByRole("tab", { name: "One" });
    const two = screen.getByRole("tab", { name: "Two" });
    const three = screen.getByRole("tab", { name: "Three" });

    expect(screen.getByRole("tablist", { name: "Demo tabs" })).toBeVisible();
    expect(one).toHaveAttribute("tabindex", "0");
    expect(two).toHaveAttribute("tabindex", "-1");

    fireEvent.keyDown(one, { key: "ArrowRight" });
    expect(two).toHaveFocus();
    expect(two).toHaveAttribute("aria-selected", "true");
    expect(one).toHaveAttribute("tabindex", "-1");

    fireEvent.keyDown(two, { key: "End" });
    expect(three).toHaveFocus();
    fireEvent.keyDown(three, { key: "ArrowRight" });
    expect(one).toHaveFocus();
    fireEvent.keyDown(one, { key: "ArrowLeft" });
    expect(three).toHaveFocus();
    fireEvent.keyDown(three, { key: "Home" });
    expect(one).toHaveFocus();

    // Anything else is left to the browser.
    fireEvent.keyDown(one, { key: "a" });
    expect(one).toHaveFocus();
  });

  it("pairs every tab with the panel it controls", () => {
    render(<Harness />);
    for (const tab of tabs) {
      const control = screen.getByRole("tab", { name: tab.label });
      const panel = document.getElementById(`demo-panel-${tab.id}`);
      expect(control).toHaveAttribute("id", `demo-tab-${tab.id}`);
      expect(control).toHaveAttribute("aria-controls", `demo-panel-${tab.id}`);
      expect(panel).toHaveAttribute("aria-labelledby", `demo-tab-${tab.id}`);
    }
    expect(screen.getByRole("tabpanel")).toHaveTextContent("One panel");

    fireEvent.click(screen.getByRole("tab", { name: "Three" }));
    expect(screen.getByRole("tabpanel")).toHaveTextContent("Three panel");
  });

  it("takes focus only where the caller asks for it", () => {
    const { unmount } = render(<Harness />);
    expect(screen.getByRole("tab", { name: "One" })).not.toHaveFocus();
    unmount();

    render(<Harness focusOnMount />);
    expect(screen.getByRole("tab", { name: "One" })).toHaveFocus();
  });
});
