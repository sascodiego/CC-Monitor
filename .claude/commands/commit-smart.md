Context-aware semantic commits:

1. ANALYZE CHANGES:
   ```bash
   git diff --name-only
   git diff --stat
   ```

2. SECTOR VALIDATION:
   - Ensure all changes within active sector
   - Flag any violations
   - Abort if cross-sector detected

3. GENERATE COMMIT MESSAGE:
   ```
   TYPE(SECTOR): Description
   
   Types: feat, fix, refactor, test, docs, style, perf
   Sector: [active sector name]
   Description: [concise, specific]
   
   Body (if needed):
   - What changed
   - Why it changed
   - Impact on sector
   ```

4. SMART COMMIT:
   ```bash
   git add -A
   git commit -m "[generated message]"
   ```

5. UPDATE TRACKING:
   - Save to GIT_CONTEXT.md
   - Update SESSION_LOG.md
   - Note in ACTIVE_CONTEXT.md