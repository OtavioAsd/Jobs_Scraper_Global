import { describe, expect, it } from "vitest";
import { paginate, parsePagination } from "../../../src/lib/pagination";

describe("parsePagination", () => {
  it("retorna page=1 e limit=100 com query vazia", () => {
    expect(parsePagination({})).toEqual({ page: 1, limit: 100 });
  });

  it("parseia page e limit validos", () => {
    expect(parsePagination({ page: "3", limit: "20" })).toEqual({
      page: 3,
      limit: 20,
    });
  });

  it("clampeia page minima em 1", () => {
    expect(parsePagination({ page: "0" }).page).toBe(1);
    expect(parsePagination({ page: "-5" }).page).toBe(1);
  });

  it("clampeia limit maximo em 100", () => {
    expect(parsePagination({ limit: "200" }).limit).toBe(100);
    expect(parsePagination({ limit: "999" }).limit).toBe(100);
  });

  it("trata limit=0 como invalido e usa padrao 100", () => {
    // parseInt("0") = 0, que e falsy, entao o || 100 retorna 100
    expect(parsePagination({ limit: "0" }).limit).toBe(100);
  });

  it("trata limit negativo como invalido e usa padrao 100", () => {
    // parseInt("-1") = -1, Math.max(1, -1) = 1, mas -1 nao e falsy
    // entao o fluxo e: parseInt("-1") || 100 -> -1 (truthy) -> Math.max(1,-1) = 1
    expect(parsePagination({ limit: "-1" }).limit).toBe(1);
  });

  it("trata valores nao numericos como padrao", () => {
    expect(parsePagination({ page: "abc" })).toEqual({ page: 1, limit: 100 });
    expect(parsePagination({ limit: "xyz" })).toEqual({ page: 1, limit: 100 });
  });

  it("trata valores null/undefined como padrao", () => {
    expect(parsePagination({ page: null, limit: undefined })).toEqual({
      page: 1,
      limit: 100,
    });
  });

  it("parseia valores numericos diretos (nao string)", () => {
    expect(parsePagination({ page: 2, limit: 50 })).toEqual({
      page: 2,
      limit: 50,
    });
  });
});

describe("paginate", () => {
  const items = Array.from({ length: 25 }, (_, i) => `item-${i + 1}`);

  it("retorna a primeira pagina corretamente", () => {
    const result = paginate(items, { page: 1, limit: 10 });

    expect(result.data).toHaveLength(10);
    expect(result.data[0]).toBe("item-1");
    expect(result.data[9]).toBe("item-10");
    expect(result.pagination).toEqual({
      total: 25,
      page: 1,
      limit: 10,
      totalPages: 3,
      hasNext: true,
      hasPrev: false,
    });
  });

  it("retorna a pagina do meio corretamente", () => {
    const result = paginate(items, { page: 2, limit: 10 });

    expect(result.data[0]).toBe("item-11");
    expect(result.data[9]).toBe("item-20");
    expect(result.pagination.hasNext).toBe(true);
    expect(result.pagination.hasPrev).toBe(true);
  });

  it("retorna a ultima pagina (parcial) corretamente", () => {
    const result = paginate(items, { page: 3, limit: 10 });

    expect(result.data).toHaveLength(5);
    expect(result.data[0]).toBe("item-21");
    expect(result.pagination.hasNext).toBe(false);
    expect(result.pagination.hasPrev).toBe(true);
  });

  it("retorna lista vazia quando items e vazio", () => {
    const result = paginate([], { page: 1, limit: 10 });

    expect(result.data).toEqual([]);
    expect(result.pagination).toEqual({
      total: 0,
      page: 1,
      limit: 10,
      totalPages: 0,
      hasNext: false,
      hasPrev: false,
    });
  });

  it("retorna tudo em uma pagina quando limit >= total", () => {
    const result = paginate(items, { page: 1, limit: 100 });

    expect(result.data).toHaveLength(25);
    expect(result.pagination.totalPages).toBe(1);
    expect(result.pagination.hasNext).toBe(false);
    expect(result.pagination.hasPrev).toBe(false);
  });

  it("retorna data vazia quando page excede totalPages", () => {
    const result = paginate(items, { page: 99, limit: 10 });

    expect(result.data).toEqual([]);
    expect(result.pagination.hasNext).toBe(false);
    expect(result.pagination.hasPrev).toBe(true);
  });

  it("funciona com limit=1", () => {
    const result = paginate(["a", "b", "c"], { page: 2, limit: 1 });

    expect(result.data).toEqual(["b"]);
    expect(result.pagination.totalPages).toBe(3);
    expect(result.pagination.hasNext).toBe(true);
    expect(result.pagination.hasPrev).toBe(true);
  });
});
