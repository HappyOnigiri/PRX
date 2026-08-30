# Documentation instructions

- Use `docs/` for durable decision history, rationale, policies, public contracts, and important constraints.
- Keep important constraints explicit when relying on inference from the code could lead to an incompatible or unsafe change.
- For details that evolve with feature work, refer to the owning implementation or generated reference instead of translating current features, schemas, or tests into prose.
- Keep each prose sentence within 200 Unicode characters.
- Split independent ideas into separate sentences, list items, or table rows instead of wrapping one long sentence across multiple lines.
- Use physical line wrapping only when a single sentence cannot be shortened or structured without losing necessary meaning.
- After applying these rules, use `make markdown-lint` to validate the remaining physical-line constraints.
- Keep structurally significant long lines in fenced code blocks or outer-pipe tables.
- Standalone indivisible tokens are also accepted.
