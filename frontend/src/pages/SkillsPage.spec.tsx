import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Outlet, Route, Routes } from "react-router-dom";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { api, type Skill } from "../lib/api";
import { SkillsPage } from "./SkillsPage";

vi.mock("../lib/api", () => ({
  api: {
    getSkills: vi.fn(),
    getProviders: vi.fn(),
  },
}));

const getSkills = vi.mocked(api.getSkills);
const getProviders = vi.mocked(api.getProviders);

function skill(name: string, extra: Partial<Skill> = {}): Skill {
  return {
    zid: name,
    name,
    slug: name,
    directoryName: name,
    rootPath: `/skills/${name}`,
    tags: [],
    status: "ready",
    lastScannedAt: "2026-08-22T00:00:00Z",
    issueCodes: [],
    conflictKinds: [],
    isConflict: false,
    isEffective: true,
    provider: { zid: "prov-1", name: "Workspace Skills", type: "workspace", rootPath: "/skills", enabled: true, priority: 1, scanMode: "recursive", lastScanStatus: "success" },
    ...extra,
  };
}

function renderPage() {
  return render(
    <MemoryRouter initialEntries={["/skills"]}>
      <Routes>
        <Route element={<Outlet context={{ refreshKey: 0 }} />}>
          <Route path="/skills" element={<SkillsPage />} />
          <Route path="/skills/:zid" element={<SkillsPage />} />
        </Route>
      </Routes>
    </MemoryRouter>,
  );
}

describe("SkillsPage search", () => {
  afterEach(() => {
    cleanup();
  });

  beforeEach(() => {
    getSkills.mockReset();
    getProviders.mockReset();
    getSkills.mockResolvedValue([skill("coding"), skill("product-development-lifecycle", { tags: ["pdl"] })]);
    getProviders.mockResolvedValue([]);
  });

  it("loads the catalog without q", async () => {
    renderPage();

    await waitFor(() => {
      expect(getSkills).toHaveBeenCalledWith({ sort: "lastScanned", grouped: true });
    });
    expect(screen.getByText("coding")).toBeInTheDocument();
  });

  it("sends q and does not keep a local text matcher", async () => {
    const user = userEvent.setup();
    getSkills.mockImplementation(async (query = {}) => {
      if (query.q) {
        return [skill("product-development-lifecycle", { tags: ["pdl"] })];
      }
      return [skill("coding"), skill("product-development-lifecycle", { tags: ["pdl"] })];
    });

    renderPage();
    await screen.findByText("coding");

    await user.type(screen.getByPlaceholderText("搜索 Skill..."), "PDL");

    await waitFor(() => {
      expect(getSkills).toHaveBeenCalledWith({ sort: "lastScanned", grouped: true, q: "PDL" });
    });
    await waitFor(() => {
      expect(screen.queryByText("coding")).not.toBeInTheDocument();
    });
    expect(screen.getByText("product-development-lifecycle")).toBeInTheDocument();
  });
});
