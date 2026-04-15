import { render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { MemoryRouter } from "react-router-dom";
import App from "./App";

function mockApiResponse(data: unknown) {
  return {
    ok: true,
    json: async () => ({ code: 0, message: "ok", data }),
  } as Response;
}

afterEach(() => {
  vi.unstubAllGlobals();
});

function installFetchMock() {
  vi.stubGlobal(
    "fetch",
    vi.fn((input: RequestInfo | URL) => {
      const url = String(input);
      if (url.includes("/api/dashboard")) {
        return Promise.resolve(
          mockApiResponse({
            providerCount: 1,
            enabledProviderCount: 1,
            skillCount: 1,
            conflictCount: 0,
            issueCount: 0,
            recentScanCount: 1,
          })
        );
      }
      if (url.includes("/api/providers")) {
        return Promise.resolve(mockApiResponse([]));
      }
      if (url.includes("/api/scan-jobs")) {
        return Promise.resolve(mockApiResponse([]));
      }
      if (url.includes("/api/issues")) {
        return Promise.resolve(mockApiResponse([]));
      }
      if (url.includes("/api/conflicts")) {
        return Promise.resolve(mockApiResponse([]));
      }
      if (url.includes("/api/skills")) {
        return Promise.resolve(mockApiResponse([]));
      }
      return Promise.resolve(mockApiResponse([]));
    })
  );
}

describe("App", () => {
  it("renders dashboard shell navigation", () => {
    installFetchMock();

    render(
      <MemoryRouter initialEntries={["/"]}>
        <App />
      </MemoryRouter>
    );

    expect(screen.getByRole("link", { name: /Skills 目录/i })).toBeInTheDocument();
    expect(screen.getByRole("link", { name: /Providers/i })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: /系统大盘/i })).toBeInTheDocument();
  });

  it("renders providers page on /providers", () => {
    installFetchMock();

    render(
      <MemoryRouter initialEntries={["/providers"]}>
        <App />
      </MemoryRouter>
    );

    expect(screen.getByText(/Providers 管理/i)).toBeInTheDocument();
  });

  it("renders Not Found on unknown path", () => {
    installFetchMock();
    render(
      <MemoryRouter initialEntries={["/does-not-exist"]}>
        <App />
      </MemoryRouter>
    );
    expect(
      screen.getByRole("heading", { name: /Not Found/i })
    ).toBeInTheDocument();
  });
});
