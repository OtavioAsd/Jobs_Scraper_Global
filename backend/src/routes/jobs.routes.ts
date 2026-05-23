import { Request, Response, Router } from "express";
import {
  cacheAbsoluteSMembers,
  cacheGetJobsByIds,
  cacheSearchKeywords,
} from "../lib/cache";
import { paginate, parsePagination } from "../lib/pagination";
import { logWarn } from "../logger";

export const jobsRoutes = Router();

/**
 * @swagger
 * /api/jobs/search:
 * get:
 * summary: Busca vagas em memória RAM no Valkey usando índices invertidos e interseção
 * tags: [Jobs]
 * parameters:
 * - in: query
 * name: keywords
 * schema:
 * type: string
 * description: 'Termos para filtrar (ex: "react,node") separados por vírgula'
 */
jobsRoutes.get("/search", async (req: Request, res: Response) => {
  try {
    const { keywords } = req.query;
    const pagination = parsePagination(req.query);

    let ids: string[] = [];
    let source = "valkey_global_index";

    if (keywords) {
      const keywordsArray = String(keywords)
        .split(",")
        .map((k) => k.trim())
        .filter(Boolean);

      ids = await cacheSearchKeywords(keywordsArray);
      source = `valkey_filtered_by_keywords:${keywordsArray.join("+")}`;
    } else {
      ids = await cacheAbsoluteSMembers("scraper:jobs:index");
    }

    const { data: pageIds, pagination: meta } = paginate(ids, pagination);
    const jobs = await cacheGetJobsByIds(pageIds);

    return res.json({
      total: meta.total,
      page: meta.page,
      limit: meta.limit,
      totalPages: meta.totalPages,
      hasNext: meta.hasNext,
      hasPrev: meta.hasPrev,
      jobs,
      source,
    });
  } catch (error) {
    logWarn("Erro ao buscar vagas no ecossistema Valkey", {
      error: (error as Error).message,
    });
    return res.status(500).json({
      message: "Erro ao recuperar vagas em memória.",
      error: (error as Error).message,
    });
  }
});
