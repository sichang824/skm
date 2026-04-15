import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { AddonsPage } from "./AddonsPage";

describe("AddonsPage", () => {
  it("renders the page title", () => {
    render(<AddonsPage />);
    expect(screen.getByText("Addons & Libraries")).toBeInTheDocument();
  });
});
