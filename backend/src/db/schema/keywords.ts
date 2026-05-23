import {
  pgTable,
  text,
  timestamp,
  uniqueIndex,
  uuid,
} from "drizzle-orm/pg-core";

import { InferInsertModel, InferSelectModel } from "drizzle-orm";

export const keywords = pgTable(
  "keywords",
  {
    id: uuid("id").defaultRandom().primaryKey(),

    keyword: text("keyword").notNull(),
    source: text("source", { enum: ["user", "scraper"] })
      .default("user")
      .notNull(),

    createdAt: timestamp("created_at").defaultNow().notNull(),
  },
  (table) => ({
    keywordUnique: uniqueIndex("keywords_keyword_unique").on(table.keyword),
  }),
);

export type Keyword = InferSelectModel<typeof keywords>;
export type NewKeyword = InferInsertModel<typeof keywords>;
