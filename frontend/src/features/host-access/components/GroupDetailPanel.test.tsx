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

function renderPanel(
  bypassSubjectCount: number,
  overrides?: Parameters<typeof createMockGroupDetailWithUsers>[0],
) {
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
          bypassSubjectCount={bypassSubjectCount}
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

describe("GroupDetailPanel — Access · read-only / bypass subjects", () => {
  it("does not show a bypass badge when the global bypass count is zero", () => {
    renderPanel(0, {
      users: [{ id: 1, username: "alice", display_name: "Alice" }],
    });

    expect(screen.queryByText(/bypass host checking/)).not.toBeInTheDocument();
  });

  it("shows the global bypass count alongside group-scoped grants", () => {
    renderPanel(3, {
      users: [{ id: 1, username: "alice", display_name: "Alice" }],
      network_policies: [{ id: 5, name: "corp-vpn", cidr: "10.0.0.0/8" }],
    });

    expect(screen.getByText("+3 bypass host checking entirely")).toBeInTheDocument();
    // Group-scoped grants remain visible alongside the bypass note
    expect(screen.getByText("Users · 1")).toBeInTheDocument();
    expect(screen.getByText("Network policies · 1")).toBeInTheDocument();
  });

  it("renders the panel for bypass subjects even with no group-scoped grants", () => {
    renderPanel(2, {
      users: [],
      network_policies: [],
    });

    expect(screen.getByText("+2 bypass host checking entirely")).toBeInTheDocument();
    expect(screen.getByText("Access · read-only")).toBeInTheDocument();
  });

  it("hides the panel entirely when there are no grants and no bypass subjects", () => {
    renderPanel(0, {
      users: [],
      network_policies: [],
    });

    expect(screen.queryByText("Access · read-only")).not.toBeInTheDocument();
  });
});
