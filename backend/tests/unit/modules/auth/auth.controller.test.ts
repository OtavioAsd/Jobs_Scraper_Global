import { Request, Response } from "express";
import { getIronSession } from "iron-session";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { z } from "zod";
import { AuthController } from "../../../../src/modules/auth/auth.controller"; // Ajuste o caminho se necessário

// Mock das dependências externas
vi.mock("iron-session", () => ({
  getIronSession: vi.fn(),
}));

// Mock dos Schemas do Zod para isolar o comportamento se necessário,
// mas vamos garantir que os inputs passem por eles perfeitamente.
vi.mock("../../../../src/types/auth.types.js", () => ({
  OAuthProviderSchema: {
    parse: vi.fn((val) => {
      if (val === "invalid") throw new z.ZodError([]);
      return val;
    }),
  },
  AuthCallbackParamsSchema: {
    parse: vi.fn((val) => {
      if (val.provider === "invalid") throw new z.ZodError([]);
      return val;
    }),
  },
}));

describe("AuthController", () => {
  let authServiceMock: any;
  let authController: AuthController;
  let reqMock: Partial<Request>;
  let resMock: Partial<Response>;
  let sessionMock: any;

  beforeEach(() => {
    vi.stubEnv("SESSION_SECRET", "um-password-longo-com-mais-de-32-caracteres");
    vi.clearAllMocks();

    // Mock do AuthService
    authServiceMock = {
      getAuthUrl: vi.fn().mockResolvedValue("https://provider.com/auth"),
      handleCallback: vi
        .fn()
        .mockResolvedValue({ token: "jwt-token", user: { id: "1" } }),
    };

    authController = new AuthController(authServiceMock);

    // Mock da Sessão do iron-session
    sessionMock = {
      save: vi.fn().mockResolvedValue(undefined),
      oauth_state: undefined,
    };
    (getIronSession as any).mockResolvedValue(sessionMock);

    // Mocks do Express Request e Response
    reqMock = {
      params: {},
      query: {},
      protocol: "http",
      get: vi.fn().mockReturnValue("localhost:3000"),
      originalUrl: "/auth/callback",
    };

    resMock = {
      json: vi.fn().mockReturnThis(),
      status: vi.fn().mockReturnThis(),
    };
  });

  describe("getUrl", () => {
    it("deve gerar a URL de autenticação com sucesso e salvar o state na sessão", async () => {
      reqMock.params = { provider: "google" };

      await authController.getUrl(reqMock as Request, resMock as Response);

      expect(sessionMock.oauth_state).toBeDefined();
      expect(typeof sessionMock.oauth_state).toBe("string");
      expect(sessionMock.save).toHaveBeenCalled();
      expect(authServiceMock.getAuthUrl).toHaveBeenCalledWith(
        "google",
        sessionMock.oauth_state,
      );
      expect(resMock.json).toHaveBeenCalledWith({
        url: "https://provider.com/auth",
      });
    });

    it("deve retornar 400 se o provider for inválido (ZodError)", async () => {
      reqMock.params = { provider: "invalid" };

      await authController.getUrl(reqMock as Request, resMock as Response);

      expect(resMock.status).toHaveBeenCalledWith(400);
      expect(resMock.json).toHaveBeenCalledWith(
        expect.objectContaining({ error: "Provider inválido" }),
      );
    });
  });

  describe("callback", () => {
    it("deve processar o callback com sucesso quando o state for válido", async () => {
      sessionMock.oauth_state = "state_secreto_123";
      reqMock.params = { provider: "google" };
      reqMock.query = { code: "auth_code_abc", state: "state_secreto_123" };

      await authController.callback(reqMock as Request, resMock as Response);

      expect(sessionMock.oauth_state).toBeUndefined(); // Deve deletar após o uso
      expect(sessionMock.save).toHaveBeenCalled();
      expect(authServiceMock.handleCallback).toHaveBeenCalledWith(
        expect.objectContaining({
          provider: "google",
          code: "auth_code_abc",
          state: "state_secreto_123",
          callbackUrl: "http://localhost:3000/auth/callback",
        }),
      );
      expect(resMock.json).toHaveBeenCalledWith({
        token: "jwt-token",
        user: { id: "1" },
      });
    });

    it("deve retornar 400 se o oauth_state estiver ausente na sessão", async () => {
      sessionMock.oauth_state = undefined;
      reqMock.params = { provider: "google" };
      reqMock.query = { code: "code", state: "any_state" };

      await authController.callback(reqMock as Request, resMock as Response);

      expect(resMock.status).toHaveBeenCalledWith(400);
      expect(resMock.json).toHaveBeenCalledWith({
        error: "OAuth state ausente",
      });
    });

    it("deve retornar 400 se o state da query for diferente do state da sessão", async () => {
      sessionMock.oauth_state = "state_original";
      reqMock.params = { provider: "google" };
      reqMock.query = { code: "code", state: "state_ataque_csrf" };

      await authController.callback(reqMock as Request, resMock as Response);

      expect(resMock.status).toHaveBeenCalledWith(400);
      expect(resMock.json).toHaveBeenCalledWith({
        error: "OAuth state inválido",
      });
    });

    it("deve retornar 500 se o serviço falhar", async () => {
      sessionMock.oauth_state = "valid_state";
      reqMock.params = { provider: "google" };
      reqMock.query = { code: "code", state: "valid_state" };

      authServiceMock.handleCallback.mockRejectedValue(
        new Error("Falha na comunicação com o provedor"),
      );

      await authController.callback(reqMock as Request, resMock as Response);

      expect(resMock.status).toHaveBeenCalledWith(500);
      expect(resMock.json).toHaveBeenCalledWith({
        error: "Falha na comunicação com o provedor",
      });
    });
  });
});
