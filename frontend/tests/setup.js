import "@testing-library/jest-dom/vitest";
import { expect, vi } from "vitest";
import * as matchers from "@testing-library/jest-dom/matchers";
import React from "react";

expect.extend(matchers);

vi.mock("react-router", () => ({
  ...vi.importActual("react-router"),
  useNavigate: () => vi.fn(),
  // Cria o elemento 'a' usando funções puras do React para o arquivo .js não quebrar
  Link: ({ children, to, ...props }) => React.createElement("a", { href: to, ...props }, children),
  NavLink: ({ children, to, ...props }) => React.createElement("a", { href: to, ...props }, children),
}));
