import { useState } from "react";
import { describe, expect, it, vi } from "vitest";
import { screen, within } from "@testing-library/react";
import { Table } from "@mantine/core";
import { GroupBadgeList } from "@/features/host-access/components/GroupBadgeList";
import { GroupColorPicker } from "@/features/host-access/components/GroupColorPicker";
import { GroupMetadataModal } from "@/features/host-access/components/GroupMetadataModal";
import { HostsTab } from "@/features/host-access/components/HostsTab";
import { IconPicker } from "@/features/host-access/components/IconPicker";
import { TombstonedHostRow } from "@/features/host-access/components/TombstonedHostRow";
import type { DraftGroup } from "@/features/host-access/drafts/hostGroupsDraft";
import { initialHostsDraft } from "@/features/host-access/drafts/knownHostsDraft";
import { createMockHost } from "@/test/mocks/data";
import { renderWithProviders, setupUser } from "@/test/utils";

function ControlledGroupColorPicker({ initial = "#4C6EF5", onChange = vi.fn() }) {
  const [value, setValue] = useState(initial);
  return (
    <GroupColorPicker
      value={value}
      onChange={(next) => {
        setValue(next);
        onChange(next);
      }}
    />
  );
}

function ControlledIconPicker({
  initial = "IconServer",
  onChange = vi.fn(),
}: {
  initial?: string | null;
  onChange?: (next: string | null) => void;
}) {
  const [value, setValue] = useState<string | null>(initial);
  return (
    <IconPicker
      value={value}
      onChange={(next) => {
        setValue(next);
        onChange(next);
      }}
      color="#4C6EF5"
    />
  );
}

function ModalResetHarness({ onSubmit = vi.fn() }) {
  const [opened, setOpened] = useState(false);
  return (
    <>
      <button type="button" onClick={() => setOpened(true)}>
        Open modal
      </button>
      <GroupMetadataModal
        opened={opened}
        onClose={() => setOpened(false)}
        initial={null}
        existingNames={[]}
        existingColors={[]}
        onSubmit={onSubmit}
      />
    </>
  );
}

const editGroup: DraftGroup = {
  id: 7,
  name: "Media",
  description: "Plex and music",
  icon: "IconMusic",
  color: "#7950F2",
  hostIds: [10],
};

describe("GroupColorPicker", () => {
  it("calls onChange when a swatch is selected", async () => {
    const user = setupUser();
    const onChange = vi.fn();
    renderWithProviders(<ControlledGroupColorPicker onChange={onChange} />);

    await user.click(screen.getByRole("button", { name: "#7950F2" }));

    expect(onChange).toHaveBeenCalledWith("#7950F2");
  });

  it("calls onChange for valid custom hex input", async () => {
    const user = setupUser();
    const onChange = vi.fn();
    renderWithProviders(<ControlledGroupColorPicker onChange={onChange} />);

    const input = screen.getByRole("textbox", { name: /custom hex colour/i });
    await user.clear(input);
    await user.type(input, "#123ABC");

    expect(onChange).toHaveBeenLastCalledWith("#123ABC");
  });

  it("shows validation for invalid hex and reverts on blur", async () => {
    const user = setupUser();
    const onChange = vi.fn();
    renderWithProviders(<ControlledGroupColorPicker initial="#4C6EF5" onChange={onChange} />);

    const input = screen.getByRole("textbox", { name: /custom hex colour/i });
    await user.clear(input);
    await user.type(input, "blue");

    expect(screen.getByText("Enter a hex colour like #4C6EF5")).toBeInTheDocument();
    expect(onChange).not.toHaveBeenCalledWith("blue");

    await user.tab();

    expect(input).toHaveValue("#4C6EF5");
    expect(screen.queryByText("Enter a hex colour like #4C6EF5")).not.toBeInTheDocument();
  });
});

describe("GroupMetadataModal", () => {
  it("shows create defaults and keeps submit disabled until a valid name is entered", () => {
    renderWithProviders(
      <GroupMetadataModal
        opened
        onClose={vi.fn()}
        initial={null}
        existingNames={[]}
        existingColors={["#4C6EF5"]}
        onSubmit={vi.fn()}
      />,
    );

    expect(screen.getByRole("dialog", { name: /new host group/i })).toBeInTheDocument();
    expect(screen.getByRole("textbox", { name: /name/i })).toHaveValue("");
    expect(screen.getByRole("textbox", { name: /description/i })).toHaveValue("");
    expect(screen.getByRole("textbox", { name: /custom hex colour/i })).toHaveValue("#7950F2");
    expect(screen.getByRole("button", { name: /create/i })).toBeDisabled();
  });

  it("shows edit defaults", () => {
    renderWithProviders(
      <GroupMetadataModal
        opened
        onClose={vi.fn()}
        initial={editGroup}
        existingNames={["Media"]}
        existingColors={["#7950F2"]}
        onSubmit={vi.fn()}
      />,
    );

    expect(screen.getByRole("dialog", { name: /edit group — media/i })).toBeInTheDocument();
    expect(screen.getByRole("textbox", { name: /name/i })).toHaveValue("Media");
    expect(screen.getByRole("textbox", { name: /description/i })).toHaveValue("Plex and music");
    expect(screen.getByRole("textbox", { name: /custom hex colour/i })).toHaveValue("#7950F2");
    expect(screen.getByRole("button", { name: "IconMusic" })).toHaveAttribute("aria-pressed", "true");
  });

  it("prevents duplicate group names", async () => {
    const user = setupUser();
    renderWithProviders(
      <GroupMetadataModal
        opened
        onClose={vi.fn()}
        initial={null}
        existingNames={["Media"]}
        existingColors={[]}
        onSubmit={vi.fn()}
      />,
    );

    await user.type(screen.getByRole("textbox", { name: /name/i }), "media");

    expect(screen.getByText("A group with this name already exists")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /create/i })).toBeDisabled();
  });

  it("submits trimmed name, nullable description, selected icon, and selected color", async () => {
    const user = setupUser();
    const onSubmit = vi.fn();
    renderWithProviders(
      <GroupMetadataModal
        opened
        onClose={vi.fn()}
        initial={null}
        existingNames={[]}
        existingColors={[]}
        onSubmit={onSubmit}
      />,
    );

    await user.type(screen.getByRole("textbox", { name: /name/i }), "  Media  ");
    await user.type(screen.getByRole("textbox", { name: /description/i }), "  Shared media hosts  ");
    await user.click(screen.getByRole("button", { name: "#F06595" }));
    await user.click(screen.getByRole("button", { name: "IconMusic" }));
    await user.click(screen.getByRole("button", { name: /create/i }));

    expect(onSubmit).toHaveBeenCalledWith({
      name: "Media",
      description: "Shared media hosts",
      icon: "IconMusic",
      color: "#F06595",
    });
  });

  it("cancel and close reset transient create state", async () => {
    const user = setupUser();
    renderWithProviders(<ModalResetHarness />);

    await user.click(screen.getByRole("button", { name: /open modal/i }));
    await user.type(screen.getByRole("textbox", { name: /name/i }), "Transient");
    await user.click(screen.getByRole("button", { name: /cancel/i }));
    await user.click(screen.getByRole("button", { name: /open modal/i }));

    expect(screen.getByRole("textbox", { name: /name/i })).toHaveValue("");
  });
});

describe("IconPicker", () => {
  it("renders selectable icon options and marks the selected option", () => {
    renderWithProviders(<ControlledIconPicker initial="IconDatabase" />);

    expect(screen.getByRole("button", { name: "IconServer" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "IconDatabase" })).toHaveAttribute("aria-pressed", "true");
  });

  it("calls onChange when selecting an icon", async () => {
    const user = setupUser();
    const onChange = vi.fn();
    renderWithProviders(<ControlledIconPicker initial="IconServer" onChange={onChange} />);

    await user.click(screen.getByRole("button", { name: "IconCloud" }));

    expect(onChange).toHaveBeenCalledWith("IconCloud");
    expect(screen.getByRole("button", { name: "IconCloud" })).toHaveAttribute("aria-pressed", "true");
  });
});

describe("TombstonedHostRow and GroupBadgeList", () => {
  it("renders a tombstoned host row with restore action", async () => {
    const user = setupUser();
    const onRestore = vi.fn();
    renderWithProviders(
      <Table>
        <Table.Tbody>
          <TombstonedHostRow host={createMockHost({ fqdn: "old.lan" })} onRestore={onRestore} />
        </Table.Tbody>
      </Table>,
    );

    const row = screen.getByRole("row");
    expect(within(row).getByText("old.lan")).toBeInTheDocument();
    expect(within(row).getByText("Will delete")).toBeInTheDocument();

    await user.click(within(row).getByRole("button", { name: /restore old\.lan/i }));

    expect(onRestore).toHaveBeenCalledOnce();
  });

  it("caps passive group chips and shows overflow count", () => {
    renderWithProviders(
      <GroupBadgeList
        groups={[
          { id: 1, name: "Media", color: "#4C6EF5", icon: "IconMusic" },
          { id: 2, name: "Databases", color: "#7950F2", icon: "IconDatabase" },
          { id: 3, name: "Very Long Operations Group", color: "#F06595", icon: "IconTool" },
          { id: 4, name: "Backups", color: "#74C0FC", icon: "IconCloud" },
          { id: 5, name: "Cameras", color: "#63E6BE", icon: "IconDeviceTv" },
        ]}
      />,
    );

    expect(screen.getByText("Media")).toBeInTheDocument();
    expect(screen.getByText("Databases")).toBeInTheDocument();
    expect(screen.getByText("Very Long Operatio…")).toBeInTheDocument();
    expect(screen.getByText("+2 more")).toBeInTheDocument();
    expect(screen.queryByText("Backups")).not.toBeInTheDocument();
  });

  it("renders clickable filter chips and reports the selected group id", async () => {
    const user = setupUser();
    const onGroupClick = vi.fn();
    renderWithProviders(
      <GroupBadgeList
        groups={[
          { id: 1, name: "Media", color: "#4C6EF5", icon: "IconMusic" },
          { id: 2, name: "Databases", color: "#7950F2", icon: "IconDatabase" },
        ]}
        selected={new Set([2])}
        onGroupClick={onGroupClick}
      />,
    );

    await user.click(screen.getByText("Databases"));

    expect(onGroupClick).toHaveBeenCalledWith(2);
  });
});

describe("HostsTab selected branch behavior", () => {
  it("renders the empty branch for an empty hosts draft", () => {
    renderWithProviders(
      <HostsTab state={initialHostsDraft()} dispatch={vi.fn()} serverGroups={[]} />,
    );

    expect(screen.getByRole("heading", { name: /no hosts yet/i })).toBeInTheDocument();
    expect(screen.getByText(/nothing is sent to the server until you click save/i)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /add host/i })).toBeInTheDocument();
  });
});
