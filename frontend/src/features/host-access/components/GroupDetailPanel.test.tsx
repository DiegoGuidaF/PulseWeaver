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

describe("GroupDetailPanel — Access · read-only", () => {
  it("shows group-scoped grants", () => {
    renderPanel({
      users: [{ id: 1, username: "alice", display_name: "Alice" }],
      network_policies: [{ id: 5, name: "corp-vpn", cidr: "10.0.0.0/8" }],
    });

    expect(screen.getByText("Access · read-only")).toBeInTheDocument();
    expect(screen.getByText("Users · 1")).toBeInTheDocument();
    expect(screen.getByText("Network policies · 1")).toBeInTheDocument();
  });

  it("hides the panel entirely when there are no grants", () => {
    renderPanel({
      users: [],
      network_policies: [],
    });

    expect(screen.queryByText("Access · read-only")).not.toBeInTheDocument();
  });
});
