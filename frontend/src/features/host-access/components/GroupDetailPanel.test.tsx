import { describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { MantineProvider } from "@mantine/core";
import { MemoryRouter } from "react-router-dom";
import { GroupDetailPanel } from "./GroupDetailPanel";
import { createMockGroupDetailWithUsers } from "@/test/mocks/data";
import type { DraftGroup, GroupsDiff } from "@/features/host-access/drafts/hostGroupsDraft";

const emptyDiff: GroupsDiff = { added: [], removed: [], changed: [], byId: new Map() };

const draftGroup: DraftGroup = {
  id: 1,
  name: "Media",
  description: null,
  icon: "server",
  color: "indigo",
  hostIds: [10],
};

function renderPanel(overrides?: Parameters<typeof createMockGroupDetailWithUsers>[0]) {
  const serverGroup = createMockGroupDetailWithUsers({
    id: 1,
    name: "Media",
    hosts: [{ id: 10, fqdn: "media.lan" }],
    ...overrides,
  });

  return render(
    <MantineProvider>
      <MemoryRouter>
        <GroupDetailPanel
          group={draftGroup}
          serverGroup={serverGroup}
          diff={emptyDiff}
          hosts={[{ id: 10, fqdn: "media.lan" }]}
          onEdit={vi.fn()}
          onDelete={vi.fn()}
          onRestore={vi.fn()}
          onToggleHost={vi.fn()}
        />
      </MemoryRouter>
    </MantineProvider>,
  );
}

describe("GroupDetailPanel — Access · read-only / bypass reach", () => {
  it("does not show a bypass badge when bypass_subject_count is zero", () => {
    renderPanel({
      users: [{ id: 1, username: "alice", display_name: "Alice" }],
      bypass_subject_count: 0,
    });

    expect(screen.queryByText(/via bypass/)).not.toBeInTheDocument();
  });

  it("shows '+N via bypass' alongside group-scoped grants when bypass reach exists", () => {
    renderPanel({
      users: [{ id: 1, username: "alice", display_name: "Alice" }],
      network_policies: [{ id: 5, name: "corp-vpn", cidr: "10.0.0.0/8" }],
      bypass_subject_count: 3,
    });

    expect(screen.getByText("+3 via bypass")).toBeInTheDocument();
    // Group-scoped grants remain visible alongside the bypass note
    expect(screen.getByText("Users · 1")).toBeInTheDocument();
    expect(screen.getByText("Network policies · 1")).toBeInTheDocument();
  });

  it("renders the panel for bypass-only reach even with no group-scoped grants", () => {
    renderPanel({
      users: [],
      network_policies: [],
      bypass_subject_count: 2,
    });

    expect(screen.getByText("+2 via bypass")).toBeInTheDocument();
    expect(screen.getByText("Access · read-only")).toBeInTheDocument();
  });

  it("hides the panel entirely when there are no grants and no bypass reach", () => {
    renderPanel({
      users: [],
      network_policies: [],
      bypass_subject_count: 0,
    });

    expect(screen.queryByText("Access · read-only")).not.toBeInTheDocument();
  });
});
