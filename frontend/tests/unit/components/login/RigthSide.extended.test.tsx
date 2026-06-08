/* eslint-disable @typescript-eslint/no-explicit-any */
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import "@testing-library/jest-dom/vitest";

vi.mock("@unpic/react", () => ({
  Image: (props: any) => <img {...props} alt={props.alt} />,
}));

vi.mock("framer-motion", () => ({
  motion: {
    div: ({ children, ...props }: any) => <div {...props}>{children}</div>,
    button: ({ children, ...props }: any) => <button {...props}>{children}</button>,
  },
}));

const mockNavigate = vi.fn();
vi.mock("react-router-dom", () => ({
  useNavigate: () => mockNavigate,
}));

const mockLogin = vi.fn();
vi.mock("@/context/AuthContext", () => ({
  useAuth: () => ({ login: mockLogin }),
}));

const mockApiGet = vi.fn();
vi.mock("@/services/api", () => ({
  api: { get: (...args: any[]) => mockApiGet(...args) },
}));

import RigthSide from "@/components/login/RigthSide";

function fillValidForm() {
  fireEvent.change(screen.getByLabelText(/email/i), { target: { value: "qa@teste.com" } });
  fireEvent.change(screen.getByLabelText(/senha/i), { target: { value: "123456" } });
}

describe("RigthSide — branches extras", () => {
  beforeEach(() => {
    mockLogin.mockReset();
    mockApiGet.mockReset();
    mockNavigate.mockReset();
  });

  // ── Branch: login com sucesso → navega para /app ─────────────────────────
  it("navega para /app após login bem-sucedido", async () => {
    mockLogin.mockResolvedValueOnce(undefined);
    render(<RigthSide />);
    fillValidForm();
    fireEvent.click(screen.getByRole("button", { name: /entrar/i }));
    await waitFor(() => expect(mockNavigate).toHaveBeenCalledWith("/app"));
  });

  // ── Branch: login falha com Axios error (response.data.error string) ──────
  it("exibe mensagem de erro do Axios quando login falha com response", async () => {
    mockLogin.mockRejectedValueOnce({
      response: { data: { error: "Credenciais inválidas" } },
    });
    render(<RigthSide />);
    fillValidForm();
    fireEvent.click(screen.getByRole("button", { name: /entrar/i }));
    await waitFor(() =>
      expect(screen.getByText("Credenciais inválidas")).toBeInTheDocument()
    );
  });

  // ── Branch: login falha sem response (erro de rede) ───────────────────────
  it("exibe erro genérico quando login falha sem response", async () => {
    mockLogin.mockRejectedValueOnce(new Error("Network Error"));
    render(<RigthSide />);
    fillValidForm();
    fireEvent.click(screen.getByRole("button", { name: /entrar/i }));
    await waitFor(() =>
      expect(screen.getByText(/credenciais inválidas ou erro de conexão/i)).toBeInTheDocument()
    );
  });

  // ── Branch: OAuth Google — sucesso com URL ────────────────────────────────
  it("redireciona para URL do Google OAuth", async () => {
    const hrefSetter = vi.fn();
    const locationSpy = vi.spyOn(window, "location", "get").mockReturnValue({
      ...window.location,
      href: "",
    } as Location);
    Object.defineProperty(window.location, "href", { set: hrefSetter, configurable: true });

    mockApiGet.mockResolvedValueOnce({ data: { url: "https://google.com/oauth" } });
    render(<RigthSide />);

    // Clica no Google pelo alt text da imagem
    fireEvent.click(screen.getByAltText(/google/i).closest("button")!);

    await waitFor(() =>
      expect(hrefSetter).toHaveBeenCalledWith("https://google.com/oauth")
    );

    locationSpy.mockRestore();
  });

  // ── Branch: OAuth — API não retorna URL → throw → globalError ─────────────
  it("exibe globalError quando OAuth não retorna URL", async () => {
    mockApiGet.mockResolvedValueOnce({ data: {} }); // sem url
    render(<RigthSide />);

    fireEvent.click(screen.getByAltText(/google/i).closest("button")!);

    await waitFor(() =>
      expect(screen.getByText(/não foi possível conectar/i)).toBeInTheDocument()
    );
  });

  // ── Branch: OAuth — erro Axios com response.data.error ───────────────────
  it("exibe mensagem do Axios no OAuth quando há response.data.error", async () => {
    mockApiGet.mockRejectedValueOnce({
      response: { data: { error: "Provedor desativado" } },
    });
    render(<RigthSide />);

    const oauthButtons = screen.getAllByRole("button").filter(
      (btn) => !btn.textContent?.match(/entrar|entrando/i)
    );
    fireEvent.click(oauthButtons[1]); // LinkedIn

    await waitFor(() =>
      expect(screen.getByText("Provedor desativado")).toBeInTheDocument()
    );
  });

  // ── Branch: OAuth — erro sem response ────────────────────────────────────
  it("exibe erro genérico de OAuth quando erro não tem response", async () => {
    mockApiGet.mockRejectedValueOnce(new Error("Network Error"));
    render(<RigthSide />);

    const oauthButtons = screen.getAllByRole("button").filter(
      (btn) => !btn.textContent?.match(/entrar|entrando/i)
    );
    fireEvent.click(oauthButtons[2]); // GitHub

    await waitFor(() =>
      expect(screen.getByText(/não foi possível conectar/i)).toBeInTheDocument()
    );
  });

  // ── Branch: isSubmitting desabilita botão de entrar ───────────────────────
  it("exibe texto 'Entrando...' e desabilita botão durante submit", async () => {
    // login nunca resolve para manter isSubmitting = true
    mockLogin.mockImplementation(() => new Promise(() => {}));
    render(<RigthSide />);
    fillValidForm();
    fireEvent.click(screen.getByRole("button", { name: /entrar/i }));
    await waitFor(() =>
      expect(screen.getByRole("button", { name: /entrando/i })).toBeDisabled()
    );
  });
});
