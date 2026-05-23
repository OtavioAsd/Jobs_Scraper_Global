import { describe, expect, it, vi } from "vitest";
import { generateUsername } from "../../../src/utils/generateUsername";

// ─── Mock do db ───────────────────────────────────────────────────────────────

function makeTx(responses: (object | null)[]) {
  let call = 0;
  return {
    query: {
      users: {
        findFirst: vi.fn(() => Promise.resolve(responses[call++] ?? null)),
      },
    },
  };
}

// ═════════════════════════════════════════════════════════════════════════════
// generateUsername
// ═════════════════════════════════════════════════════════════════════════════

describe("generateUsername", () => {
  it("retorna o base normalizado quando não existe conflito", async () => {
    const tx = makeTx([null]);
    const result = await generateUsername("João Silva", tx as any);
    expect(result).toBe("joaosilva");
  });

  it("retorna 'user' quando base é undefined", async () => {
    const tx = makeTx([null]);
    const result = await generateUsername(undefined, tx as any);
    expect(result).toBe("user");
  });

  it("retorna 'user' quando base normaliza para string vazia", async () => {
    const tx = makeTx([null]);
    const result = await generateUsername("!@#$%", tx as any);
    expect(result).toBe("user");
  });

  it("adiciona sufixo _1 quando base já existe", async () => {
    // primeira query (base) conflita, segunda (_1) está livre
    const tx = makeTx([{ id: "existing" }, null]);
    const result = await generateUsername("admin", tx as any);
    expect(result).toBe("admin_1");
  });

  it("incrementa sufixo até encontrar username livre", async () => {
    // base, _1, _2 conflitam; _3 está livre
    const tx = makeTx([{ id: "1" }, { id: "2" }, { id: "3" }, null]);
    const result = await generateUsername("dev", tx as any);
    expect(result).toBe("dev_3");
  });

  it("normaliza acentos e caracteres especiais", async () => {
    const tx = makeTx([null]);
    const result = await generateUsername("Ângela Ção", tx as any);
    expect(result).toBe("angelacao");
  });

  it("remove espaços", async () => {
    const tx = makeTx([null]);
    const result = await generateUsername("john doe", tx as any);
    expect(result).toBe("johndoe");
  });

  it("remove caracteres não alfanuméricos exceto underscore", async () => {
    const tx = makeTx([null]);
    const result = await generateUsername("user.name-test", tx as any);
    expect(result).toBe("usernametest");
  });

  it("trunca base em 20 caracteres", async () => {
    const tx = makeTx([null]);
    const result = await generateUsername(
      "abcdefghijklmnopqrstuvwxyz",
      tx as any,
    );
    expect(result).toBe("abcdefghijklmnopqrst");
    expect(result.length).toBe(20);
  });

  it("retorna sufixo aleatório após 20 tentativas esgotadas", async () => {
    // base + _1 até _20 todos conflitam (21 respostas com objeto)
    const responses = Array.from({ length: 21 }, () => ({ id: "x" }));
    const tx = makeTx(responses);

    const result = await generateUsername("taken", tx as any);

    expect(result).toMatch(/^taken_\d+$/);
    // sufixo aleatório está entre 0 e 9999
    const suffix = parseInt(result.split("_")[1]);
    expect(suffix).toBeGreaterThanOrEqual(0);
    expect(suffix).toBeLessThan(10000);
  });

  it("preserva underscores no base", async () => {
    const tx = makeTx([null]);
    const result = await generateUsername("my_user", tx as any);
    expect(result).toBe("my_user");
  });

  it("converte para minúsculas", async () => {
    const tx = makeTx([null]);
    const result = await generateUsername("MyUser", tx as any);
    expect(result).toBe("myuser");
  });
});
